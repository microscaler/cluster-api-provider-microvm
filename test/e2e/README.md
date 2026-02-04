# End-to-end testing

The e2e tests are designed to be run in a number of ways.
If you are developing CAPMVM, continue reading this doc.
If you are testing the whole of Liquid Metal, check out the [Liquid Metal Acceptance Tests][lmats]
docs.

## Start flintlock

You will need a running flintlock server. This can be done locally, if you are
working on Linux. Mac or windows users have the option to run flintlock on
an [Equinix][equinix] host, but if you do not have an account there you will not
be able to run these tests.

```bash
git clone https://github.com/liquidmetal-dev/flintlock
cd flintlock
sudo ./hack/scripts/provision.sh --grpc-address 0.0.0.0:9090 --dev --insecure
```

This will clone flintlock and bootstrap your machine to run a server. You can
read the [flintlock docs][fl-docs] if you would like to set this up manually
and see each individual step. **Start flintlock with `--grpc-endpoint` (or `--grpc-address`) set to `0.0.0.0:9090`.** Listening on `0.0.0.0` ensures any client (Kind, CAPMVM, or another host on your network) can reach it, regardless of the host’s IP. Binding to a specific IP would only work for that one host.

If you ran the above command, flintlock will be running as a `systemd` service.

## DHCP

When microvms are created they will request an IP from a DHCP server. Your router
should have one, but you may need to check the settings or start a new server
for the purpose of the tests.

TODO: explain this

## Required tools

Ensure you have the following installed:
- [kind](https://kind.sigs.k8s.io/)
- [docker](https://docs.docker.com/engine/install/ubuntu/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [clusterctl](https://cluster-api.sigs.k8s.io/user/quick-start.html#install-clusterctl)
- [kustomize](https://kustomize.io/) (used by the CAPI test framework to build provider manifests)

Optional, for inspecting Flintlock and MicroVMs during or after a run:
- **hammertime** or **grpcurl** to list MicroVMs: `hammertime list` or `grpcurl -plaintext -d '{}' 127.0.0.1:9090 microvm.services.api.v1alpha1.MicroVM/ListMicroVMs` (the `fl` CLI does not have a list command).
- [grpcurl](https://github.com/fullstorydev/grpcurl) – call gRPC from the command line (e.g. list MicroVMs against real Flintlock or the mock).

For Flintlock host setup and troubleshooting (containerd, devmapper, macvtap, NAT, images), see [hack/README.md](../../hack/README.md). For an audit, healthcheck script and Ansible playbook, see [BOOTSTRAP_AUDIT.md](BOOTSTRAP_AUDIT.md).

## Run the tests

The CAPMVM manager image used in e2e (`ghcr.io/liquidmetal-dev/cluster-api-provider-microvm:e2e`) is **built locally** by `make e2e` (via `docker-build` with `TAG=e2e`) and is not published to the registry. The test loads it into Kind from your Docker daemon. Do not run the e2e test binary directly without building the image first (e.g. run `make e2e` which builds then runs).

You can either use an existing flintlock server or start one in a container.

**Option 1 – Use a real flintlock server** (no mock; generated templates use only the host(s) you pass):

Use the `e2e-with-flintlock` or `e2e-with-flintlock-retain-artifacts` targets. They require `-e2e.flintlock-hosts` and do **not** add the mock, so the cluster template is filled only with your flintlock server address(es). You can pass the host (and optional VIP) via **env vars** so you don’t need to type `E2E_ARGS` every time:

```bash
# Using env vars (no E2E_ARGS needed)
export E2E_FLINTLOCK_HOSTS=192.168.1.57:9090
export E2E_CAPMVM_VIP=172.18.0.1   # optional; use Kind gateway when e2e runs on same host
make e2e-with-flintlock-retain-artifacts

# Or inline
make e2e-with-flintlock-retain-artifacts E2E_FLINTLOCK_HOSTS=192.168.1.57:9090 E2E_CAPMVM_VIP=172.18.0.1
```

Or pass everything in `E2E_ARGS`:

```bash
make e2e-with-flintlock E2E_ARGS="-e2e.flintlock-hosts <host>:9090 -e2e.capmvm.vip-address=<vip>"
make e2e-with-flintlock-retain-artifacts E2E_ARGS="-e2e.flintlock-hosts <host>:9090 -e2e.capmvm.vip-address=<vip>"
```

You can also use the generic `make e2e` and pass `-e2e.flintlock-hosts` (and omit `-e2e.use-flintlock-mock`); the dedicated targets above enforce that the mock is not used.

**Option 2 – Use the flintlock API mock** (no external server; mocks the flintlock gRPC API in-process):

```bash
make e2e-with-flintlock-mock
```

This runs an in-process mock of the flintlock MicroVM gRPC service so e2e can exercise CAPMVM without a real flintlock/Firecracker stack. The mock implements CreateMicroVM, GetMicroVM, DeleteMicroVM, ListMicroVMs and ListMicroVMsStream with in-memory state. The mock listens on `0.0.0.0` so it is reachable from any client. **On Linux you must set `E2E_FLINTLOCK_HOST`** to the address that Kind nodes use to reach the host (e.g. the Kind network gateway or the docker bridge IP); see “Control plane VIP and networking” for how to discover it.

_Note: the tests will default to using `192.168.1.25` for the workload cluster's
control plane VIP. You will need to verify that this is within your network and
unused, and **reachable from the Kind cluster** (where the CAPI controller runs), or
configure the tests to use another by setting the `-e2e.capmvm.vip-address`
flag. See "Test options" and "Control plane VIP and networking" below._

The tests will take ~5 mins to run.

They will:
- Create a new kind cluster
- Init the cluster with CAPI providers
- Generate a CAPMVM workload cluster
- Apply the cluster
- Wait for the control plane and the worker nodes to start
- Verify that all given flintlock hosts were used (if you can start a second
	flintlock server on a different port, you can pass both to the tests with
	`make e2e E2E_ARGS="-e2e.flintlock-hosts 1.2.3.4:9090,4.5.6.7:9091"`)
- Deploy nginx to the workload cluster
- Ensure that nginx starts successfully
- Delete the workload cluster
- Delete the kind cluster

To speed up your testing cycle, you can pass the `-e2e.existing-cluster` flag.
See "Test options" below for details.

### Where the tutorial “generate + apply” runs

The [Liquid Metal “Create a Liquid Metal cluster”](https://liquidmetal.dev/docs/tutorial-basics/create) steps map to our e2e as follows:

| Tutorial | Our e2e code |
|----------|----------------|
| `clusterctl generate cluster -i microvm:$CAPMVM_VERSION -f cilium $CLUSTER_NAME > cluster.yaml` | **`clusterctl.ConfigCluster(...)`** in `test/e2e/utils/clusterctl.go` (lines 56–72). Same idea: generate a cluster template from the microvm provider and cilium flavor, with cluster name and machine counts. We get the YAML in memory and optionally dump it to the artefact dir. |
| `kubectl apply -f cluster.yaml` | **`clusterctl.ApplyCustomClusterTemplateAndWait(...)`** in `test/e2e/utils/clusterctl.go` (lines 93–103). This applies the generated template to the management cluster and then waits for the workload cluster to be ready. |

Both of these run **inside the `It()` specs**, when **`utils.ApplyClusterTemplateAndWait(ctx, input)`** is called:

- **v1alpha1 flow:** `test/e2e/e2e_test.go` line **96** (right after the `By("Creating microvm cluster (v1alpha1 only)")` step).
- **v1alpha2 flow:** `test/e2e/e2e_test.go` line **169** (right after the `By("Creating microvm cluster (v1alpha2 only)")` step).

**Execution order:** Ginkgo runs **BeforeSuite** first (`mngr.Setup()` in `test/e2e/e2e_suite_test.go`). That boots Kind, runs `clusterctl init`, and waits for the CAPMVM webhook. Only after BeforeSuite succeeds does Ginkgo run the `Describe` / `It()` blocks. So:

- If the suite **fails in BeforeSuite** (e.g. “capmvm-controller-manager is not ready after 5m”), the **generate and apply steps never run**. You get a ready Kind management cluster in some runs, but the test exits before “Creating microvm cluster” and no workload cluster is applied, so the controller and Flintlock never see a Cluster/MicrovmCluster.
- If you see “nothing in the logs” for provisioning, check whether the failure is in BeforeSuite (look for the last `STEP:` or failure message). If it is, fix the management-cluster readiness so the `It()` that calls `ApplyClusterTemplateAndWait` actually runs.

## Checking Flintlock and MicroVMs

To verify what Flintlock has created (or what the mock is tracking), use the same endpoint the tests use.

**With a real Flintlock server** – use the same host and port as `-e2e.flintlock-hosts` (e.g. `localhost:9090` when Flintlock runs on the same machine). The service name `microvm.services.api.v1alpha1.MicroVM` and method `ListMicroVMs` are the real API from the [Flintlock](https://github.com/liquidmetal-dev/flintlock) proto (`api/services/microvm/v1alpha1/microvms.proto`).

**Note:** `ListMicroVMs` reads from containerd's content store. You need **containerd running** and Flintlock started with the correct `--containerd-socket`. Otherwise you get:
`getting all microvms: ... walking content store: unknown service containerd.services.content.v1.Content: not implemented`

To list MicroVMs (use **hammertime** or **grpcurl**; `fl` does not have a list command):

```bash
# Using hammertime
hammertime list

# Or with grpcurl (install: go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest)
grpcurl -plaintext localhost:9090 list
grpcurl -plaintext -d '{}' localhost:9090 microvm.services.api.v1alpha1.MicroVM/ListMicroVMs
```

**Fix:** Start containerd and run Flintlock with the right `--containerd-socket` (see [Flintlock docs](https://github.com/liquidmetal-dev/flintlock)). If you used the Flintlock [provision script](https://github.com/liquidmetal-dev/flintlock/blob/main/hack/scripts/provision.sh) with `--dev`, it creates a separate **containerd-dev** instance (socket `/run/containerd-dev/containerd.sock`); ensure `containerd-dev.service` is running, not only `containerd.service`. Or use the e2e mock (below) to list MicroVMs without containerd.

**With the flintlock mock** – the test logs the address when it starts, for example:

```
STEP: Flintlock mock listening at <host>:<port>
```

Here `<host>` is from `E2E_FLINTLOCK_HOST` on Linux, or `host.docker.internal` on Darwin/Windows.

Use that host:port from your machine (use `localhost` if the mock is on the same host, or the printed IP). To list MicroVMs use **hammertime list** or **grpcurl** (the `fl` CLI does not have a list command). Against the mock, use **grpcurl** because the mock only implements the MicroVM gRPC API, not containerd:

```bash
# List services (mock has gRPC reflection)
grpcurl -plaintext localhost:45105 list

# List MicroVMs created during the test
grpcurl -plaintext -d '{}' localhost:45105 microvm.services.api.v1alpha1.MicroVM/ListMicroVMs
```

If the test exits, the mock process and its in-memory state are gone. Use `-e2e.skip-cleanup` and keep the test running (or inspect while the suite is still running) to check state.

## Test options

The following flags are available:

```
Usage of /home/claudia/workspace/cluster-api-provider-microvm/test/e2e/e2e.test:
  -e2e.artefact-dir string
        Location to store test yamls, logs, etc. (default "/home/claudia/workspace/cluster-api-provider-microvm/test/e2e/_artefacts")
  -e2e.capmvm.kubernetes-version string
        Version of k8s to run in the workload cluster(s) (default "1.23.10"; must be >= 1.22.0 for CAPI kubeadm-bootstrap)
  -e2e.capmvm.vip-address string
        Address for the kubevip load balancer. (default "192.168.1.25")
  -e2e.config string
        Path to e2e config for this suite. (default "config/e2e_conf_v1beta2.yaml")
        Use config/e2e_conf.yaml for v1beta1 (not supported with current CAPI test deps).
  -e2e.existing-cluster
        If true, uses the current context for the management cluster and will not create a new one.
  -e2e.flintlock-hosts value
        Comma separated list of addresses to flintlock servers. eg '1.2.3.4:9090,5.6.7.8:9090'
  -e2e.use-flintlock-mock
        Run an in-process mock of the flintlock gRPC API (no -e2e.flintlock-hosts needed).
  -e2e.skip-cleanup
        Do not delete test-created workload clusters or the management kind cluster.
```

These can be passed to the `make` command:

```bash
make e2e E2E_ARGS="-e2e.skip-cleanup"
```

To use the `e2e.existing-cluster` boolean flag, you will need to ensure that the
cluster is set as the current context.

### Connecting to Kind clusters and switching context

When you have multiple Kind clusters (e.g. `lm-management`, `test-3amv0x`, `test-r7dweu` from e2e runs with `-e2e.skip-cleanup`), you can list them and switch context as follows.

**List Kind clusters:**
```bash
kind get clusters
```

**Use the current e2e management cluster (written by Ginkgo as it runs):**

The e2e suite writes an env file to the artefact dir as soon as the management cluster is created. Use it to point at that run’s cluster:

```bash
# When using make e2e, artefacts are in ~/flintlock/_artefacts (see Makefile TEST_ARTEFACTS)
source ~/flintlock/_artefacts/e2e-management-cluster.env
export KUBECONFIG="$E2E_MANAGEMENT_KUBECONFIG"
kubectl config use-context "$E2E_MANAGEMENT_CONTEXT"
kubectl get nodes
```

If `kubectl` is not in your PATH (e.g. on the host where e2e runs), you can run it inside the Kind node using the cluster name from the env file:

```bash
source ~/flintlock/_artefacts/e2e-management-cluster.env
docker exec ${E2E_MANAGEMENT_CLUSTER_NAME}-control-plane kubectl get nodes
docker exec ${E2E_MANAGEMENT_CLUSTER_NAME}-control-plane kubectl get pods -A
```

The file (e.g. `~/flintlock/_artefacts/e2e-management-cluster.env` when using make e2e) defines:
- `E2E_MANAGEMENT_KUBECONFIG` – path to that run’s Kind kubeconfig
- `E2E_MANAGEMENT_CONTEXT` – kubeconfig context (e.g. `kind-test-3amv0x`)
- `E2E_MANAGEMENT_CLUSTER_NAME` – Kind cluster name (e.g. `test-3amv0x`)

**Switch to a different Kind cluster by name:**

Use a cluster name from `kind get clusters` (e.g. `test-i08lrp`, `test-3amv0x`). Do not use the literal `<name>`—replace it with your cluster name.

**Dynamic: use the first (or last) cluster from the list:**

```bash
# First cluster in the list
CLUSTER=$(kind get clusters | head -1)
# Or last (often the most recent): CLUSTER=$(kind get clusters | tail -1)

kind export kubeconfig --name "$CLUSTER"
kubectl config use-context "kind-$CLUSTER"
```

To use that cluster in the current shell only (without changing your default kubeconfig):

```bash
CLUSTER=$(kind get clusters | head -1)
export KUBECONFIG=$(mktemp)
kind export kubeconfig --name "$CLUSTER" --kubeconfig "$KUBECONFIG"
kubectl config use-context "kind-$CLUSTER"
```

**Static: use a specific cluster name** (replace `test-i08lrp` with a name from `kind get clusters`):

```bash
kind export kubeconfig --name test-i08lrp
kubectl config use-context kind-test-i08lrp
```

**List contexts (all clusters in your current KUBECONFIG):**
```bash
kubectl config get-contexts
```

### CAPI contract version (v1beta1 vs v1beta2)

The default e2e config uses the **v1beta2** contract (CAPI v1.11.x) so that it matches the CAPI test framework in go.mod (clusterctl v1.11.x only supports v1beta2 management clusters).

| Make target | Config | CAPI version | Contract |
|-------------|--------|-------------|----------|
| `make e2e`, `make e2e-with-flintlock-mock`, `make e2e-v1beta2` | `config/e2e_conf_v1beta2.yaml` | v1.11.1 | v1beta2 (default) |
| `make e2e-v1beta1` | `config/e2e_conf.yaml` | v1.1.5 | v1beta1 (unsupported with current test deps) |

Examples:

```bash
# v1beta2 with real flintlock (no mock; templates use only -e2e.flintlock-hosts)
make e2e-with-flintlock E2E_ARGS="-e2e.flintlock-hosts <host>:9090 -e2e.capmvm.vip-address=<vip>"
make e2e-with-flintlock-retain-artifacts E2E_ARGS="-e2e.flintlock-hosts <host>:9090 -e2e.capmvm.vip-address=<vip>"

# v1beta2 with flintlock mock
make e2e-with-flintlock-mock
make e2e-with-flintlock-mock E2E_ARGS="-e2e.skip-cleanup"   # keep Kind cluster and workload after run

# Generic e2e (pass either flintlock-hosts or use-flintlock-mock)
make e2e E2E_ARGS="-e2e.flintlock-hosts $FL:9090"
make e2e-v1beta2 E2E_ARGS="-e2e.flintlock-hosts $FL:9090"

# v1beta1 (fails with current CAPI test framework v1.11.1)
make e2e-v1beta1 E2E_ARGS="-e2e.flintlock-hosts $FL:9090"
```

To override the config via the flag: `make e2e E2E_ARGS="-e2e.config=config/e2e_conf_v1beta2.yaml -e2e.flintlock-hosts $FL:9090"`.

_Note: `-e2e.flintlock-hosts` and `-e2e.artefact-dir` are already passed to the
tests as part of the `make` command._

## Control plane VIP and networking

The workload cluster’s API server is reached at a single **control plane VIP** (set by `-e2e.capmvm.vip-address`, default `192.168.1.25`). Two separate concerns matter: **reachability from the management cluster** and **binding on the workload nodes**.

### Reachability from the management cluster (Kind)

The **CAPI controller runs inside the Kind cluster** (in a pod). It must be able to open TCP connections to `CONTROL_PLANE_VIP:6443` to check workload cluster health and apply add-ons. So the VIP must be an IP that **Kind nodes (Docker containers) can route to**.

- **E2e and Kind on the same Linux host**: Use the host IP that the **Kind network** uses as gateway so pods can reach the host. Kind creates a custom bridge (e.g. `br-*` for network `kind`); the gateway is often **Kind gateway** (not `docker0`’s docker bridge). Check with `docker network inspect kind` and use the gateway as VIP and for flintlock if needed:
  ```bash
  export E2E_FLINTLOCK_HOST=$(docker network inspect kind --format '{{range .IPAM.Config}}{{.Gateway}}{{end}}')
  make e2e E2E_ARGS="-e2e.flintlock-hosts <flintlock-host>:9090 -e2e.capmvm.vip-address=$E2E_FLINTLOCK_HOST"
  ```
  Ensure nothing else uses that IP; kube-vip on the workload control plane nodes will bind to it (see below).

- **E2e on Mac, Kind in Docker Desktop, Flintlock on a remote Linux host**: Kind nodes run in Docker on the Mac. The VIP must be an IP reachable from those containers. Options: use the Mac’s IP on the LAN (if Docker Desktop routes to it), or run Kind on the same Linux host as Flintlock and use that host’s Docker bridge IP as above.

- **Default VIP**: The default is a single placeholder address; from inside Kind it may be unreachable (no route). If you see `no route to host` or `connection to the workload cluster is down` in CAPI controller logs, override with a VIP reachable from Kind (e.g. the Kind network gateway; see above).

### Binding the VIP on workload nodes

When you use **real Flintlock**, control plane MicroVMs boot and kube-vip runs on them. The kube-vip manifest is configured with `--address ${CONTROL_PLANE_VIP}` and binds that IP on the node’s default route interface. So:

- The VIP must be an IP that the **workload nodes** can use on their network (e.g. same subnet as the host or a dedicated VIP range).
- If you set the VIP to the **host’s Docker or Kind bridge gateway** so Kind can reach it, ensure that the MicroVMs’ default route goes through an interface that can have that address. Otherwise kube-vip may fail to bind.

For **local dev** with Kind and Flintlock on one Linux host, using the Kind network gateway for the VIP is the usual setup so both “Kind → VIP” and “workload node → bind VIP” work.

### Validating networking on the host

On the Flintlock host: `ss -tlnp | grep 9090` (flintlock on `*:9090`) and `grpcurl -plaintext 127.0.0.1:9090 list` (should list `microvm.services.api.v1alpha1.MicroVM`). From Kind: ensure the host (or VIP) is reachable on 9090 and 6443. For full host checks (flintlock, containerd, KVM, parent interface), see [hack/README.md](../../hack/README.md).

### E2E config alignment with the Linux host

When running e2e against a real Flintlock host, ensure the following match.

| Item | E2E default / config | Host must have |
|------|----------------------|----------------|
| **Kubernetes version** | `1.23.10` (default in code and templates; must be >= 1.22.0) | Root image tag matches: `capmvm-kubernetes:1.23.10` in containerd `flintlock` namespace |
| **Root image** | `ghcr.io/liquidmetal-dev/capmvm-kubernetes:1.23.10` | Pulled in `flintlock` namespace (see "Pre-pulling images" in Troubleshooting) |
| **Kernel image** | `ghcr.io/liquidmetal-dev/flintlock-kernel:5.10.77` | Pulled in `flintlock` namespace |
| **Network type** | `tap` or `macvtap` in cluster template `networkInterfaces[].type` | **tap**: host bridge + `--bridge-name`; upstream Firecracker. **macvtap**: host parent interface + `--parent-iface`; Liquid Metal Firecracker fork. See [hack/README.md](../../hack/README.md#network-interface-options-tap-and-macvtap). |
| **Flintlock port** | Passed via `-e2e.flintlock-hosts <host>:9090` | Flintlock listening on **`0.0.0.0:9090`** so any client (Kind, remote host) can reach it |
| **VIP** | Override with `-e2e.capmvm.vip-address` | When Kind runs on the same host as Flintlock, use the **Kind network gateway** so CAPI in Kind can reach the workload API |

No changes are required in `config/e2e_conf_v1beta2.yaml` (or `e2e_conf.yaml`) for the host above; image versions live in the cluster templates and in code defaults. Pass the correct flags when invoking e2e:

```bash
# Example: e2e and Kind on the same host as Flintlock (no specific IP; use your host or Kind gateway)
FL=$(docker network inspect kind --format '{{range .IPAM.Config}}{{.Gateway}}{{end}}')
make e2e E2E_ARGS="-e2e.flintlock-hosts ${FL}:9090 -e2e.capmvm.vip-address=$FL"
```

## Troubleshooting

### "capmvm-controller-manager is not ready" or "failed to connect to the management cluster" in BeforeSuite

`clusterctl init` waits for the CAPMVM controller manager deployment to become ready. The manager only becomes ready after cert-manager has issued webhook certificates and the webhook server is up. On slow environments this can exceed the wait timeout.

- **Use a longer wait**: The e2e configs set `default/wait-controllers` and `bootstrap/wait-controllers` to `15m`. Ensure you are using the intended config (e.g. `config/e2e_conf_v1beta2.yaml`).
- **Inspect after a failure**: Run with `-e2e.skip-cleanup` so the Kind cluster is left in place when the suite fails. Then:

  ```bash
  export KUBECONFIG=<path-to-kind-kubeconfig>   # e.g. from test artefacts or ~/.kube/kind-config-<clusterName>
  kubectl get pods -n capmvm-system
  kubectl get pods -n cert-manager
  kubectl logs -n capmvm-system deployment/capmvm-controller-manager -c manager --tail=200
  kubectl describe certificate -n capmvm-system
  ```

  Check for certificate not ready, image pull errors, or controller startup errors. Increasing `initialDelaySeconds` for the manager readiness probe (see `config/default/manager_probes.yaml`) can help if the first probe runs before the webhook is listening.

### Ginkgo fails when applying the cluster template (workload cluster)

The e2e flow is similar to the [Liquid Metal “Create a Liquid Metal cluster” tutorial](https://liquidmetal.dev/docs/tutorial-basics/create): after the management cluster (Kind) is up and CAPI is running, the test runs the equivalent of generating a cluster manifest and applying it. If that apply step fails:

- **Ensure flintlock is configured**: You must pass either `-e2e.flintlock-hosts <host:port>[, ...]` or `-e2e.use-flintlock-mock`. The template’s `MicrovmCluster.spec.placement.staticPool.hosts` is filled from this; without it the test would inject an empty list and the webhook may reject it (at least one host required).
- **Ensure CONTROL_PLANE_VIP is reachable from Kind**: The test sets this from `-e2e.capmvm.vip-address` (default `192.168.1.25`). The cluster template uses it for the control plane endpoint and kube-vip. The **CAPI controller runs inside the Kind cluster** (in a pod), so it must be able to dial `CONTROL_PLANE_VIP:6443` from there. If you see `connection to the workload cluster is down` or `connect: no route to host` in `capi-controller-manager` logs, the VIP is not routable from Kind. Use an IP that Kind nodes can reach (e.g. the Kind network gateway; see “Control plane VIP and networking”), and pass it with `-e2e.capmvm.vip-address`.
- **See the exact apply error**: Run with `-e2e.skip-cleanup` so the management cluster remains. Template dumps are written under `<artefact-dir>` (with `make e2e`, that is `~/flintlock/_artefacts/`):
  - **`cluster-template-raw.yaml`** – written as soon as the template is generated (before flintlock hosts are added). If you see this but not `cluster-template.yaml`, the failure is in “Adding provided flintlock hosts” (e.g. template structure changed).
  - **`cluster-template.yaml`** – written after flintlock hosts are injected, right before apply. Use this to reproduce the apply error.
  If **neither** file exists, the failure was before the “Creating microvm cluster” step (e.g. BeforeSuite or “Getting the cluster template yaml”).
  To reproduce an apply failure:
  ```bash
  export KUBECONFIG=<path-to-kind-kubeconfig>
  # When using make e2e (artefacts in ~/flintlock/_artefacts):
  kubectl apply -f ~/flintlock/_artefacts/cluster-template.yaml
  ```
  Check `kubectl get events -A --sort-by='.lastTimestamp'` and any webhook/validation errors. For webhook issues, check `kubectl logs -n capmvm-system deployment/capmvm-controller-manager -c manager`.
- **Compare with the tutorial**: The [tutorial](https://liquidmetal.dev/docs/tutorial-basics/create) uses `clusterctl generate cluster` then edits the manifest. The e2e templates use **macvtap** by default; for host setup (bridge vs macvtap, Firecracker fork) see [hack/README.md](../../hack/README.md).

### "connection to the workload cluster is down" / "no route to host" in capi-controller-manager

The CAPI controller runs **inside the Kind (management) cluster**. It connects to the workload cluster’s API server at `CONTROL_PLANE_VIP:6443` (default `192.168.1.25:6443`). Those errors mean the controller cannot reach that address from inside Kind.

- **With `-e2e.use-flintlock-mock`**: No real MicroVMs or control plane nodes are created. Nothing ever listens on the VIP, so CAPI will never successfully connect. Repeated `connection to the workload cluster is down` and later `no route to host` (or connection timeouts) in `capi-controller-manager` logs are **expected**. The test may still pass if it only asserts that Cluster/Machine resources are created and that the infrastructure provider reports ready; it may fail at a step that requires the workload cluster to be reachable (e.g. deploying add-ons like Cilium or checking workload cluster nodes).
- **With real Flintlock**: The VIP must be an IP that **Kind nodes (Docker containers) can route to**. Use the Kind network gateway (see “Control plane VIP and networking”) and pass it with `-e2e.capmvm.vip-address`. Ensure that when real control plane nodes come up, kube-vip is configured to use that same IP.

Pod and service CIDR **ranges** in the cluster template (e.g. `POD_CIDR`, `SERVICES_CIDR`) are unrelated to this; they apply inside the workload cluster. The issue here is the single **control plane endpoint IP** (VIP), not a range.

### "Bootstrap secret is not ready" / Machines stuck in Pending (real Flintlock)

CAPMVM only creates a MicroVM once **bootstrap data** exists: the CAPI Machine must have `spec.bootstrap.dataSecretName` set by the kubeadm-bootstrap controller. So you see:

- **CAPMVM logs**: repeated `"Bootstrap secret is not ready"` for every MicrovmMachine.
- **CAPI logs**: `"Waiting for bootstrap provider to generate data secret"` and `"Waiting for infrastructure provider to create machine infrastructure"`.
- **Cluster**: Phase=Provisioned, CP Available=0, no Machines becoming Ready.

This is a **dependency chain**: kubeadm-bootstrap must set the secret for the **first control plane** machine → CAPMVM creates that MicroVM → machine boots and runs kubeadm init → control plane becomes available. For the **first** control plane node, the CAPI kubeadm-bootstrap provider should generate **init** (not join) data without waiting for the cluster API to be reachable. If it instead waits for "Cluster control plane to be initialized" or "control plane to be available", you get a deadlock.

**Checks:**

1. **KubeadmConfig for the control plane machine** (replace namespace and name with your cluster’s):
   ```bash
   kubectl get kubeadmconfig -n <workload-cluster-namespace>
   kubectl describe kubeadmconfig -n <ns> <control-plane-machine-name>
   ```
   Look at **DataSecretCreated** and **status.conditions**. If the message is "Waiting for Cluster control plane to be initialized" for the **first** CP machine, the bootstrap provider is treating it as join or is blocked on the endpoint. That behaviour is in the CAPI kubeadm-bootstrap controller (upstream).

2. **Cluster conditions**: `kubectl get cluster -n <ns> <name> -o yaml` and check `status.conditions` and `status.initialization`.

3. **With real Flintlock**: once bootstrap data exists, CAPMVM will call Flintlock (CreateMicroVM). Ensure Flintlock and containerd (and the flintlock namespace images) are correct on the host (see [hack/README.md](../../hack/README.md)).

4. **Kubernetes version**: The CAPI kubeadm-bootstrap provider (v1.11+) does not support Kubernetes versions below 1.22.0. If you see *"the bootstrap provider for kubeadm doesn't support Kubernetes version lower than v1.22.0"* in `capi-kubeadm-bootstrap-controller-manager` logs, ensure the cluster template and `-e2e.capmvm.kubernetes-version` use 1.22.x or higher (e2e default is 1.23.10). Pre-pull the matching root image (e.g. `capmvm-kubernetes:1.23.10`) in the flintlock containerd namespace.

5. **"macvtap network interfaces not supported"**: If CAPMVM logs show *"creating microvm: macvtap network interfaces not supported by the microvm provider"*, the **microvm provider (Firecracker) does not support macvtap**. Upstream Firecracker does not; you need either (a) **Liquid Metal’s Firecracker fork** with macvtap support (build from the `feature/macvtap` branch or use a release from [liquidmetal-dev/firecracker](https://github.com/liquidmetal-dev/firecracker)), or (b) **use tap** in the cluster templates (change `type: "macvtap"` to `type: "tap"` and set up a bridge and `--bridge-name` on the host). For host setup (parent-iface, macvlan), see [hack/README.md](../../hack/README.md).

The "connection refused" at `172.18.0.1:6443` is expected until the first control plane MicroVM is up and kube-vip (or the API server) is listening on the VIP. Progress is: bootstrap secret for first CP → CAPMVM creates first CP MicroVM → VM boots → kubeadm init and kube-vip → 6443 reachable.

### Linux host (containerd, flintlock, devmapper, images)

When e2e or a real cluster uses a remote Flintlock host, run the relevant checks on that host (containerd, flintlockd, images in `flintlock` namespace, KVM, Firecracker, parent interface, devmapper if you see "snapshotter not loaded: devmapper"). Full steps: [hack/README.md](../../hack/README.md).

### "unrecognized format \"int32\"" / "\"int64\"" during clusterctl init

Those **INFO** lines come from the **Kubernetes API server** inside the Kind cluster (apiextensions-apiserver), not from CAPMVM or clusterctl. When CRDs are applied, the API server validates OpenAPI schema formats and logs a warning for formats it doesn’t recognize; it doesn’t support the standard OpenAPI `int32`/`int64` formats in that check, so you see the message even though the types are valid. **They are harmless** and do not affect behaviour.

To fix the warning upstream (so future Kubernetes/Kind versions don’t log it), the change belongs in [kubernetes/kubernetes](https://github.com/kubernetes/kubernetes) in `staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/validation/formats.go`: add `int32` and `int64` to the supported versioned formats (e.g. in the first `versionedFormats` entry’s `formats` set). See `hack/patches/README-unrecognized-format.md` in this repo for the exact patch and how to contribute it.

[lmats]: https://github.com/liquidmetal-dev/liquid-metal-acceptance-tests
[equinix]: https://metal.equinix.com/
