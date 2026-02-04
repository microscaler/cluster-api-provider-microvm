//go:build e2e
// +build e2e

package e2e_test

import (
	"context"
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/liquidmetal-dev/cluster-api-provider-microvm/test/e2e/utils"
	"k8s.io/utils/pointer"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api/test/framework"
	"sigs.k8s.io/cluster-api/test/framework/clusterctl"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CAPMVM e2e suite. Two separate flows: one stands up a v1alpha1-only workload cluster,
// the other stands up a v1alpha2-only workload cluster. There is no mixing of API
// versions within a single cluster.
var _ = Describe("CAPMVM", func() {
	var (
		ctx           context.Context
		cancelWatches context.CancelFunc

		namespace *corev1.Namespace
	)

	BeforeEach(func() {
		ctx = mngr.Ctx()
	})

	AfterEach(func() {
		if !mngr.SkipCleanup {
			framework.DeleteAllClustersAndWait(ctx, framework.DeleteAllClustersAndWaitInput{
				Namespace: namespace.Name,
			}, mngr.Cfg.GetIntervals("default", "wait-delete-cluster")...)
		}

		if cancelWatches != nil {
			cancelWatches()
		}
	})

	// Flow 1: v1alpha1 cluster only. This flow creates a single workload cluster
	// using only infrastructure.cluster.x-k8s.io/v1alpha1 (MicrovmCluster, MicrovmMachine, etc.).
	Describe("v1alpha1 cluster flow", func() {
		It("should create cluster with single control plane node and 5 worker nodes", func() {
			specName := "simple-cluster-v1alpha1"

			namespace, cancelWatches = framework.CreateNamespaceAndWatchEvents(ctx, framework.CreateNamespaceAndWatchEventsInput{
				Creator:   mngr.ClusterProxy.GetClient(),
				ClientSet: mngr.ClusterProxy.GetClientSet(),
				Name:      fmt.Sprintf("%s-%s", specName, util.RandomString(6)),
				LogFolder: filepath.Join(mngr.ArtefactDir, "logs", "clusters", mngr.ClusterProxy.GetName()),
			})

			By("Creating microvm cluster (v1alpha1 only)")
			clusterName := fmt.Sprintf("%s-%s", specName, util.RandomString(6))

			utils.SetEnvVar("CONTROL_PLANE_VIP", mngr.VIPAddress, false)
			utils.SetEnvVar("KUBERNETES_VERSION", fmt.Sprintf("v%s", mngr.KubernetesVersion), false)
			utils.SetEnvVar("MVM_ROOT_IMAGE", fmt.Sprintf("%s:%s", "ghcr.io/liquidmetal-dev/capmvm-kubernetes", mngr.KubernetesVersion), false)

			result := &clusterctl.ApplyClusterTemplateAndWaitResult{}

			input := utils.ApplyClusterInput{
				Hosts: mngr.FlintlockHosts,
				Input: clusterctl.ApplyClusterTemplateAndWaitInput{
					ClusterProxy: mngr.ClusterProxy,
					ConfigCluster: clusterctl.ConfigClusterInput{
						LogFolder:                filepath.Join(mngr.ArtefactDir, "logs", "clusters", mngr.ClusterProxy.GetName()),
						ClusterctlConfigPath:     mngr.ClusterctlCfg,
						KubeconfigPath:           mngr.KubeconfigPath,
						InfrastructureProvider:   utils.GetMicrovmProviderName(mngr.Cfg),
						Flavor:                   utils.Cilium,
						Namespace:                namespace.Name,
						ClusterName:              clusterName,
						KubernetesVersion:        fmt.Sprintf("v%s", mngr.KubernetesVersion),
						ControlPlaneMachineCount: pointer.Int64Ptr(1),
						WorkerMachineCount:       pointer.Int64Ptr(5),
					},
					WaitForClusterIntervals:      mngr.Cfg.GetIntervals(specName, "wait-cluster"),
					WaitForControlPlaneIntervals: mngr.Cfg.GetIntervals(specName, "wait-control-plane"),
					WaitForMachineDeployments:    mngr.Cfg.GetIntervals(specName, "wait-worker-nodes"),
				},
				Result: result,
			}

			utils.ApplyClusterTemplateAndWait(ctx, input)

			By("Checking that microvms are allocated across all given flintlock hosts")
			Expect(utils.FailureDomainSpread(mngr.ClusterProxy, namespace.Name, clusterName)).To(Equal(len(mngr.FlintlockHosts)),
				"Nodes were not distributed across all flintlock hosts.")

			By("Checking that an application can be deployed to the workload cluster")
			var depReplicas int32 = 2
			depName := "nginx-deployment"
			depNamespace := "default"

			nginx := utils.Nginx(depName, depNamespace, depReplicas)
			workloadClient := mngr.ClusterProxy.GetWorkloadCluster(ctx, namespace.Name, clusterName).GetClient()
			Expect(workloadClient.Create(ctx, nginx)).To(Succeed())

			Eventually(func() bool {
				created := &appsv1.Deployment{}
				err := workloadClient.Get(ctx, client.ObjectKey{Namespace: depNamespace, Name: depName}, created)
				Expect(err).NotTo(HaveOccurred())
				return created.Status.ReadyReplicas == depReplicas
			}, mngr.Cfg.GetIntervals("default", "wait-workload-task")...).Should(BeTrue())

			By("PASSED!")
		})
	})

	// Flow 2: v1alpha2 cluster only. This flow creates a single workload cluster
	// using only infrastructure.cluster.x-k8s.io/v1alpha2 (MicrovmCluster, MicrovmMachine, etc.).
	// v1alpha1 and v1alpha2 are never mixed in the same cluster.
	Describe("v1alpha2 cluster flow", func() {
		It("should create cluster with single control plane node and 5 worker nodes", func() {
			specName := "simple-cluster-v1alpha2"

			namespace, cancelWatches = framework.CreateNamespaceAndWatchEvents(ctx, framework.CreateNamespaceAndWatchEventsInput{
				Creator:   mngr.ClusterProxy.GetClient(),
				ClientSet: mngr.ClusterProxy.GetClientSet(),
				Name:      fmt.Sprintf("%s-%s", specName, util.RandomString(6)),
				LogFolder: filepath.Join(mngr.ArtefactDir, "logs", "clusters", mngr.ClusterProxy.GetName()),
			})

			By("Creating microvm cluster (v1alpha2 only)")
			clusterName := fmt.Sprintf("%s-%s", specName, util.RandomString(6))

			utils.SetEnvVar("CONTROL_PLANE_VIP", mngr.VIPAddress, false)
			utils.SetEnvVar("KUBERNETES_VERSION", fmt.Sprintf("v%s", mngr.KubernetesVersion), false)
			utils.SetEnvVar("MVM_ROOT_IMAGE", fmt.Sprintf("%s:%s", "ghcr.io/liquidmetal-dev/capmvm-kubernetes", mngr.KubernetesVersion), false)

			result := &clusterctl.ApplyClusterTemplateAndWaitResult{}

			input := utils.ApplyClusterInput{
				Hosts: mngr.FlintlockHosts,
				Input: clusterctl.ApplyClusterTemplateAndWaitInput{
					ClusterProxy: mngr.ClusterProxy,
					ConfigCluster: clusterctl.ConfigClusterInput{
						LogFolder:                filepath.Join(mngr.ArtefactDir, "logs", "clusters", mngr.ClusterProxy.GetName()),
						ClusterctlConfigPath:     mngr.ClusterctlCfg,
						KubeconfigPath:           mngr.KubeconfigPath,
						InfrastructureProvider:   utils.GetMicrovmProviderName(mngr.Cfg),
						Flavor:                   utils.V1Alpha2Cilium,
						Namespace:                namespace.Name,
						ClusterName:              clusterName,
						KubernetesVersion:        fmt.Sprintf("v%s", mngr.KubernetesVersion),
						ControlPlaneMachineCount: pointer.Int64Ptr(1),
						WorkerMachineCount:       pointer.Int64Ptr(5),
					},
					WaitForClusterIntervals:      mngr.Cfg.GetIntervals(specName, "wait-cluster"),
					WaitForControlPlaneIntervals: mngr.Cfg.GetIntervals(specName, "wait-control-plane"),
					WaitForMachineDeployments:    mngr.Cfg.GetIntervals(specName, "wait-worker-nodes"),
				},
				Result: result,
			}

			utils.ApplyClusterTemplateAndWait(ctx, input)

			By("Checking that microvms are allocated across all given flintlock hosts")
			Expect(utils.FailureDomainSpread(mngr.ClusterProxy, namespace.Name, clusterName)).To(Equal(len(mngr.FlintlockHosts)),
				"Nodes were not distributed across all flintlock hosts.")

			By("Checking that an application can be deployed to the workload cluster")
			var depReplicas int32 = 2
			depName := "nginx-deployment"
			depNamespace := "default"

			nginx := utils.Nginx(depName, depNamespace, depReplicas)
			workloadClient := mngr.ClusterProxy.GetWorkloadCluster(ctx, namespace.Name, clusterName).GetClient()
			Expect(workloadClient.Create(ctx, nginx)).To(Succeed())

			Eventually(func() bool {
				created := &appsv1.Deployment{}
				err := workloadClient.Get(ctx, client.ObjectKey{Namespace: depNamespace, Name: depName}, created)
				Expect(err).NotTo(HaveOccurred())
				return created.Status.ReadyReplicas == depReplicas
			}, mngr.Cfg.GetIntervals("default", "wait-workload-task")...).Should(BeTrue())

			By("PASSED!")
		})
	})
})
