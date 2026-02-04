//go:build e2e
// +build e2e

package utils

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
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

func (m *EnvManager) bootstrapLocalKind() {
	kubeconfigPath := ""

	if !m.UseExistingCluster {
		m.pullImages()

		bootInput := bootstrap.CreateKindBootstrapClusterAndLoadImagesInput{
			Name:               m.Cfg.ManagementClusterName,
			RequiresDockerSock: m.Cfg.HasDockerProvider(),
			Images:             m.Cfg.Images,
			LogFolder:          m.ArtefactDir + "/logs/bootstrap",
		}
		m.ClusterProvider = bootstrap.CreateKindBootstrapClusterAndLoadImages(m.ctx, bootInput)

		kubeconfigPath = m.ClusterProvider.GetKubeconfigPath()
	}

	m.ClusterProxy = framework.NewClusterProxy("bootstrap", kubeconfigPath, DefaultScheme())
	m.KubeconfigPath = kubeconfigPath
}

// This can go once we have bumped cluster-api to 1.2.0.
// bootstrap.CreateKindBootstrapClusterAndLoadImages will do this for us then.
func (m *EnvManager) pullImages() {
	By("Pulling management cluster images")

	containerRuntime, err := container.NewDockerClient()
	Expect(err).NotTo(HaveOccurred())

	for _, image := range m.Cfg.Images {
		if strings.Contains(image.Name, "microvm:e2e") {
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

// startFlintlockMock runs an in-process mock of the flintlock gRPC API and sets
// m.Params.FlintlockHosts so the CAPMVM controller in Kind can reach it. Kind nodes
// reach the host via host.docker.internal (Docker Desktop) or 172.17.0.1 (Linux).
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
	m.Params.FlintlockHosts = []string{net.JoinHostPort(host, port)}
	m.flintlockMockStop = stop
	By(fmt.Sprintf("Flintlock mock listening at %s", m.Params.FlintlockHosts[0]))
}

// flintlockHostAddress returns the address that Kind nodes use to reach the host.
func flintlockHostAddress() string {
	if h := os.Getenv("E2E_FLINTLOCK_HOST"); h != "" {
		return h
	}
	switch runtime.GOOS {
	case "darwin", "windows":
		return "host.docker.internal"
	default:
		return "172.17.0.1" // docker0 bridge; override with E2E_FLINTLOCK_HOST if needed
	}
}
