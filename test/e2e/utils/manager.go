//go:build e2e
// +build e2e

package utils

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/bootstrap"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/test/infrastructure/container"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"gopkg.in/yaml.v2"
)

// EnvManager holds various objects required for running the tests and the test
// environment.
type EnvManager struct {
	*Params
	Cfg             *clusterctl.E2EConfig
	ClusterProxy    framework.ClusterProxy
	ClusterProvider bootstrap.ClusterProvider
	ClusterctlCfg   string
	KubeconfigPath  string

	// flintlockMockStop is set when we start the flintlock mock for teardown.
	flintlockMockStop func()
	ctx               context.Context //nolint: containedctx // don't care.
}

// NewEnvManager returns a new EnvManager.
func NewEnvManager(p *Params) *EnvManager {
	return &EnvManager{
		Params: p,
		ctx:    context.TODO(),
	}
}

// Setup will prepare a local Kind cluster as a CAPI management cluster.
func (m *EnvManager) Setup() {
	// Set controller-runtime logger so clusterctl and its k8s client (e.g. warning
	// handler) do not panic with "log.SetLogger(...) was never called".
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	// Process the command line flags, fail fast if wrong.
	m.validateInput()

	// Optionally start the flintlock gRPC mock so tests have an endpoint without a real flintlock server.
	if m.UseFlintlockMock {
		m.startFlintlockMock()
	}

	// Read the file at m.E2EConfigPath to discover providers, versions, yamls, etc
	// to be used in this test.
	m.loadE2EConfig()
	Expect(m.Cfg).NotTo(BeNil())

	// Create a new kind cluster (if UseExistingCluster is not set) and load the
	// required images.
	// Set a ClusterProxy which can be used to interact with the resulting cluster.
	m.bootstrapLocalKind()
	Expect(m.ClusterProxy).NotTo(BeNil())
	if !m.UseExistingCluster {
		Expect(m.KubeconfigPath).NotTo(BeNil())
	}

	// Create a directory for clusterctl to store configuration yamls etc.
	m.createClusterctlRepo()
	Expect(m.ClusterctlCfg).NotTo(Equal(""))

	// Use clusterctl to init the kind cluster with CAPI and CAPMVM controllers.
	// After this the management cluster is ready to accept creation of workload
	// clusters.
	m.initKindCluster()

	// Wait for the CAPMVM webhook to have a ready endpoint so that apply of
	// cluster templates does not hit "connection refused" (webhook can lag
	// behind deployment availability).
	m.waitForCapmvmWebhookReady()

	// Give the API server time to resolve and cache the CRD conversion webhook
	// (capmvm-webhook-service). If we apply a cluster template immediately,
	// conversion can fail with "the server could not find the requested resource"
	// because the apiserver's webhook client was created when the CRD was applied
	// (before the service was ready).
	m.waitForConversionWebhookSettled()
}

// Teardown will delete the local kind management cluster and remove any
// related artefacts.
func (m *EnvManager) Teardown() {
	if m.flintlockMockStop != nil {
		m.flintlockMockStop()
		m.flintlockMockStop = nil
	}
	if !m.SkipCleanup && !m.UseExistingCluster {
		if m.ClusterProvider != nil {
			m.ClusterProvider.Dispose(m.ctx)
		}

		if m.ClusterProxy != nil {
			m.ClusterProxy.Dispose(m.ctx)
		}
	}
}

// Ctx returns the EnvManager's context so that it can be used throughout the
// suite.
func (m *EnvManager) Ctx() context.Context {
	return m.ctx
}

func (m *EnvManager) validateInput() {
	By(fmt.Sprintf("Validating test params: %#v", m.Params))
	if !m.UseFlintlockMock {
		Expect(m.FlintlockHosts).ToNot(HaveLen(0), "At least one address for a flintlock server is required (or use -e2e.use-flintlock-mock).")
	}
	Expect(m.E2EConfigPath).To(BeAnExistingFile(), "A valid path to a clusterctl.E2EConfig is required.")
	Expect(m.ArtefactDir).NotTo(Equal(""), "A valid path for the test artefacts folder is required.")
}

func (m *EnvManager) loadE2EConfig() {
	m.Cfg = clusterctl.LoadE2EConfig(m.ctx, clusterctl.LoadE2EConfigInput{
		ConfigPath: m.E2EConfigPath,
	})
}

// capmvmE2EImage is the manager image used in e2e. It is built locally (make e2e does
// docker-build with TAG=e2e) and is not published to the registry; we load it into Kind ourselves.
const capmvmE2EImage = "ghcr.io/liquidmetal-dev/cluster-api-provider-microvm:e2e"

func (m *EnvManager) bootstrapLocalKind() {
	kubeconfigPath := ""

	if !m.UseExistingCluster {
		m.pullImages()

		// Exclude the CAPMVM e2e image from the list: the CAPI framework pulls then loads,
		// and ghcr.io/.../cluster-api-provider-microvm:e2e is not in the registry (only dev, v0.x exist).
		// We load the locally built image into Kind after the cluster is created.
		images := filterOutImage(m.Cfg.Images, capmvmE2EImage)

		bootInput := bootstrap.CreateKindBootstrapClusterAndLoadImagesInput{
			Name:               m.Cfg.ManagementClusterName,
			RequiresDockerSock: m.Cfg.HasDockerProvider(),
			Images:             images,
			LogFolder:          m.ArtefactDir + "/logs/bootstrap",
		}
		m.ClusterProvider = bootstrap.CreateKindBootstrapClusterAndLoadImages(m.ctx, bootInput)

		kubeconfigPath = m.ClusterProvider.GetKubeconfigPath()

		// Load the locally built CAPMVM e2e image into Kind (built by make e2e via docker-build).
		m.loadCapmvmE2EImageIntoKind()
	}

	m.ClusterProxy = framework.NewClusterProxy("bootstrap", kubeconfigPath, DefaultScheme())
	m.KubeconfigPath = kubeconfigPath

	if !m.UseExistingCluster && kubeconfigPath != "" {
		m.writeE2EManagementClusterEnv(kubeconfigPath)
	}
}

// writeE2EManagementClusterEnv writes an env file to the artefact dir so you can
// source it to get KUBECONFIG and context for the e2e management cluster (e.g. after -e2e.skip-cleanup).
func (m *EnvManager) writeE2EManagementClusterEnv(kubeconfigPath string) {
	cfg, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return // best-effort; don't fail the test
	}
	currentContext := cfg.CurrentContext
	clusterName := currentContext
	if strings.HasPrefix(currentContext, "kind-") {
		clusterName = strings.TrimPrefix(currentContext, "kind-")
	}
	envPath := filepath.Join(m.ArtefactDir, "e2e-management-cluster.env")
	content := fmt.Sprintf("# E2E management cluster (Kind); source this to use: source %s\n", envPath) +
		fmt.Sprintf("export E2E_MANAGEMENT_KUBECONFIG=%q\n", kubeconfigPath) +
		fmt.Sprintf("export E2E_MANAGEMENT_CONTEXT=%q\n", currentContext) +
		fmt.Sprintf("export E2E_MANAGEMENT_CLUSTER_NAME=%q\n", clusterName) +
		"# To use: export KUBECONFIG=\"$E2E_MANAGEMENT_KUBECONFIG\" && kubectl config use-context \"$E2E_MANAGEMENT_CONTEXT\"\n"
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		return
	}
	By(fmt.Sprintf("Wrote management cluster env to %s (cluster name: %s)", envPath, clusterName))
}

func filterOutImage(images []clusterctl.ContainerImage, name string) []clusterctl.ContainerImage {
	out := make([]clusterctl.ContainerImage, 0, len(images))
	for _, img := range images {
		if img.Name != name {
			out = append(out, img)
		}
	}
	return out
}

// loadCapmvmE2EImageIntoKind loads the locally built ghcr.io/.../cluster-api-provider-microvm:e2e
// image into the Kind cluster. The image is built by make e2e (docker-build with TAG=e2e) and
// is not published to the registry.
func (m *EnvManager) loadCapmvmE2EImageIntoKind() {
	By("Loading CAPMVM e2e image into Kind cluster")
	cmd := exec.CommandContext(m.ctx, "kind", "load", "docker-image", capmvmE2EImage, "--name", m.Cfg.ManagementClusterName)
	cmd.Stdout = nil
	cmd.Stderr = nil
	Expect(cmd.Run()).To(Succeed(), "kind load docker-image %s failed (ensure the image is built locally, e.g. make e2e or make docker-build TAG=e2e)", capmvmE2EImage)
}

// This can go once we have bumped cluster-api to 1.2.0.
// bootstrap.CreateKindBootstrapClusterAndLoadImages will do this for us then.
func (m *EnvManager) pullImages() {
	By("Pulling management cluster images")

	containerRuntime, err := container.NewDockerClient()
	Expect(err).NotTo(HaveOccurred())

	for _, image := range m.Cfg.Images {
		if image.Name == capmvmE2EImage {
			continue
		}

		By("Pulling image " + image.Name)
		Expect(containerRuntime.PullContainerImageIfNotExists(m.ctx, image.Name)).To(Succeed())
	}
}

// patchClusterctlConfigMicrovmURL rewrites the clusterctl config so the microvm provider
// URL points at the explicit version directory (e.g. v0.6.99) instead of "latest", avoiding
// clusterctl's "latest" resolution which can fail with "failed to find releases tagged with a valid semantic version number".
func (m *EnvManager) patchClusterctlConfigMicrovmURL(repoFolder string) {
	var microvmVersion string
	for _, p := range m.Cfg.Providers {
		if p.Type == string(clusterctlv1.InfrastructureProviderType) && p.Name == "microvm" && len(p.Versions) > 0 {
			microvmVersion = p.Versions[0].Name
			break
		}
	}
	if microvmVersion == "" {
		return
	}
	data, err := os.ReadFile(m.ClusterctlCfg)
	Expect(err).NotTo(HaveOccurred())
	var cfg map[string]interface{}
	Expect(yaml.Unmarshal(data, &cfg)).To(Succeed())
	providers, _ := cfg["providers"].([]interface{})
	if providers == nil {
		return
	}
	for _, p := range providers {
		pm, _ := p.(map[interface{}]interface{})
		if pm == nil {
			continue
		}
		if name, _ := pm["name"].(string); name != "microvm" {
			continue
		}
		urlStr, _ := pm["url"].(string)
		if urlStr == "" {
			continue
		}
		// Replace .../infrastructure-microvm/latest/components.yaml with .../v0.6.99/...
		// Use forward slashes to match URLs in the config file.
		oldSuffix := "infrastructure-microvm/latest/components.yaml"
		newSuffix := "infrastructure-microvm/" + microvmVersion + "/components.yaml"
		urlStr = strings.TrimSuffix(urlStr, oldSuffix) + newSuffix
		pm["url"] = urlStr
		break
	}
	out, err := yaml.Marshal(cfg)
	Expect(err).NotTo(HaveOccurred())
	Expect(os.WriteFile(m.ClusterctlCfg, out, 0600)).To(Succeed())
}

func (m *EnvManager) createClusterctlRepo() {
	// Use an absolute path so clusterctl's local repository client can resolve "latest" to a version
	// (it lists version subdirs under RepositoryFolder/providerLabel; relative paths can break resolution).
	repoFolder := m.ArtefactDir + "/clusterctl"
	if abs, err := filepath.Abs(repoFolder); err == nil {
		repoFolder = abs
	}
	m.ClusterctlCfg = clusterctl.CreateRepository(m.ctx,
		clusterctl.CreateRepositoryInput{
			E2EConfig:        m.Cfg,
			RepositoryFolder: repoFolder,
		})
	// Patch the microvm provider URL to use the explicit version instead of "latest".
	// clusterctl's local repo client can fail to resolve "latest" (e.g. "failed to find releases tagged with a valid semantic version number");
	// pointing directly at the version dir (e.g. v0.6.99) avoids that resolution.
	m.patchClusterctlConfigMicrovmURL(repoFolder)
}

// waitForCapmvmWebhookReady waits until the capmvm-webhook-service has at least
// one ready endpoint, so that admission/validation webhooks can be reached when
// applying cluster templates.
func (m *EnvManager) waitForCapmvmWebhookReady() {
	By("Waiting for CAPMVM webhook service to have a ready endpoint")
	client := m.ClusterProxy.GetClient()
	nn := types.NamespacedName{Namespace: "capmvm-system", Name: "capmvm-webhook-service"}
	Eventually(func() bool {
		ep := &corev1.Endpoints{}
		err := client.Get(m.ctx, nn, ep)
		if err != nil {
			return false
		}
		for _, sub := range ep.Subsets {
			if len(sub.Addresses) > 0 {
				return true
			}
		}
		return false
	}, 2*time.Minute, 2*time.Second).Should(BeTrue(), "CAPMVM webhook service never got a ready endpoint")
}

// waitForConversionWebhookSettled waits so the API server has time to resolve
// the CRD conversion webhook (capmvm-webhook-service). The apiextensions-apiserver
// resolves the webhook when handling CRDs; if the service was not ready at that
// time, conversion can fail with "the server could not find the requested resource".
func (m *EnvManager) waitForConversionWebhookSettled() {
	By("Waiting for API server to resolve CRD conversion webhook")
	time.Sleep(5 * time.Second)
}

func (m *EnvManager) initKindCluster() {
	logFolder := m.ArtefactDir + "/logs"
	if abs, err := filepath.Abs(logFolder); err == nil {
		logFolder = abs
	}
	initInput := clusterctl.InitManagementClusterAndWatchControllerLogsInput{
		ClusterProxy:            m.ClusterProxy,
		ClusterctlConfigPath:    m.ClusterctlCfg,
		BootstrapProviders:      providers(m.Cfg, clusterctlv1.BootstrapProviderType),
		ControlPlaneProviders:   providers(m.Cfg, clusterctlv1.ControlPlaneProviderType),
		InfrastructureProviders: m.Cfg.InfrastructureProviders(),
		LogFolder:               logFolder,
	}
	clusterctl.InitManagementClusterAndWatchControllerLogs(m.ctx, initInput,
		m.Cfg.GetIntervals(m.ClusterProxy.GetName(), "wait-controllers")...)
}

func providers(cfg *clusterctl.E2EConfig, providerType clusterctlv1.ProviderType) []string {
	pList := []string{}

	for _, provider := range cfg.Providers {
		if provider.Type == string(providerType) {
			pList = append(pList, provider.Name)
		}
	}

	return pList
}

// GetMicrovmProviderName returns the microvm infrastructure provider name with version
// (e.g. "microvm:v0.6.99") from the e2e config. It must match the version in
// test/e2e/config/e2e_conf*.yaml so clusterctl.ConfigCluster can resolve the provider.
func GetMicrovmProviderName(cfg *clusterctl.E2EConfig) string {
	for _, p := range cfg.Providers {
		if p.Type == string(clusterctlv1.InfrastructureProviderType) && p.Name == "microvm" && len(p.Versions) > 0 {
			return p.Name + ":" + p.Versions[0].Name
		}
	}
	return "microvm:v0.6.99" // fallback if config shape changes
}

// startFlintlockMock runs an in-process mock of the flintlock gRPC API and sets
// m.Params.FlintlockHosts so the CAPMVM controller in Kind can reach it. The mock
// listens on 0.0.0.0 so it is reachable from any client. Kind nodes reach the host
// via host.docker.internal (Darwin/Windows) or via E2E_FLINTLOCK_HOST (Linux).
func (m *EnvManager) startFlintlockMock() {
	By("Starting flintlock gRPC mock for e2e")
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	Expect(err).NotTo(HaveOccurred())

	mock := NewFlintlockMock()
	stop, err := mock.ServeGRPC(m.ctx, listener)
	Expect(err).NotTo(HaveOccurred())

	_, port, err := net.SplitHostPort(listener.Addr().String())
	Expect(err).NotTo(HaveOccurred())
	host := flintlockHostAddress()
	Expect(host).NotTo(BeEmpty(),
		"on Linux set E2E_FLINTLOCK_HOST to the address Kind uses to reach the host (e.g. Kind network gateway or docker0); see test/e2e/README.md")
	m.Params.FlintlockHosts = []string{net.JoinHostPort(host, port)}
	m.flintlockMockStop = stop
	By(fmt.Sprintf("Flintlock mock listening at %s", m.Params.FlintlockHosts[0]))
}

// flintlockHostAddress returns the address that Kind nodes use to reach the host.
// No specific IP is hardcoded: on Darwin/Windows use host.docker.internal; on Linux
// E2E_FLINTLOCK_HOST must be set (e.g. Kind network gateway or docker bridge).
func flintlockHostAddress() string {
	if h := os.Getenv("E2E_FLINTLOCK_HOST"); h != "" {
		return h
	}
	switch runtime.GOOS {
	case "darwin", "windows":
		return "host.docker.internal"
	default:
		return "" // Linux: set E2E_FLINTLOCK_HOST so Kind can reach the mock
	}
}
