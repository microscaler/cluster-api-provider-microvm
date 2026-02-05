# Sample Custom Resources

Complete CR samples are grouped by API version. Use the version that matches your Cluster API Provider MicroVM (CAPMVM) and Cluster API versions.

## Prerequisites

Before using these samples, ensure:

- **A Cluster API management cluster exists** – e.g. created with [clusterctl and kind](https://cluster-api.sigs.k8s.io/user/quick-start.html) (or another bootstrap method).
- **Optionally, a target cluster** – if you are [pivoting](https://cluster-api.sigs.k8s.io/tasks/cluster-lifecycle/cluster-pivoting.html) from a bootstrap cluster into a dedicated management cluster, that target control-plane cluster should be available and the provider (including CAPMVM) installed there.
- **Linux hosts for MicroVMs** – the endpoints in your `MicrovmCluster.spec.placement.staticPool.hosts` must be machines that run [Flintlock](https://github.com/liquidmetal-dev/flintlock) and [Firecracker](https://github.com/firecracker-microvm/firecracker). This repo includes an [Ansible playbook](../../hack/ansible/README.md) to set up such hosts (containerd, devmapper, Flintlock drop-in, image pre-pull). See **hack/ansible/** for usage and variables.

These samples assume you are applying them in a cluster where the CAPMVM provider and its CRDs are already installed.

| Directory   | API version | Notes |
|------------|-------------|--------|
| `v1alpha1/` | `infrastructure.cluster.x-k8s.io/v1alpha1` | Deprecated; removal planned August 2026. Prefer v1alpha2. |
| `v1alpha2/` | `infrastructure.cluster.x-k8s.io/v1alpha2` | Current; use for new clusters. |

Each version directory contains:

- **microvmcluster.yaml** – Infrastructure cluster (reference from `Cluster.spec.infrastructureRef`).
- **microvmmachinetemplate.yaml** – Machine template (reference from `KubeadmControlPlane.machineTemplate.infrastructureRef` or `MachineDeployment.template.spec.infrastructureRef`).

For full cluster definitions (Cluster + KubeadmControlPlane + MachineDeployment + bootstrap), see the root **templates/** directory (e.g. `cluster-template.yaml`).
