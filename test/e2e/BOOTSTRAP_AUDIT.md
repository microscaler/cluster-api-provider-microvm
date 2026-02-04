# Bootstrap audit: Flintlock host setup and e2e readiness

This document audits everything done to get a Linux host (Flintlock/containerd host) ready for CAPMVM e2e tests, including code touch points, troubleshooting steps, and automation (healthcheck script and Ansible playbook).

## 1. Code touch points

### 1.1 Repo files created or modified for host bootstrap / e2e

| Path | Purpose |
|------|--------|
| **test/e2e/README.md** | E2e runbook: how to run tests, options, VIP/networking, short troubleshooting with pointers to hack/README. |
| **hack/README.md** | Host setup and troubleshooting: macvtap/parent-iface, containerd/flintlock, pre-pull images, devmapper (Connect and setup devmapper), NAT-over-WiFi, Docker/bridge, other checks. Links to healthcheck and Ansible. |
| **hack/containerd-config-dev.toml** | Containerd config for a separate **containerd-dev** instance with devmapper snapshotter (pool `flintlock-dev-thinpool`). Deploy to `/etc/containerd/config-dev.toml` on the host. |
| **hack/containerd-dev.service** | Systemd unit for containerd-dev. Deploy to `/etc/systemd/system/containerd-dev.service`. |
| **test/e2e/BOOTSTRAP_AUDIT.md** | This audit: code touch points, troubleshooting, healthcheck, Ansible. |
| **test/e2e/bootstrap-healthcheck.py** | Python script that checks all host aspects (services, thinpool, images, KVM, Firecracker, gRPC). Run on the Flintlock host (as root) or remotely (limited checks). |
| **hack/ansible/** | Ansible playbook and vars to set up a new Linux host (deps, thinpool, containerd-dev, Flintlock drop-in, images). |

### 1.2 E2e code that depends on host behaviour

| Path | Relevance |
|------|-----------|
| **test/e2e/utils/params.go** | `-e2e.flintlock-hosts` (comma-separated host:port), `-e2e.capmvm.vip-address`, `-e2e.capmvm.kubernetes-version`. |
| **test/e2e/utils/defaults.go** | `DefaultVIPAddress` (e.g. 192.168.1.25), `DefaultKubernetesVersion` (1.23.10). Override VIP when Kind and Flintlock are on same/different networks. |
| **test/e2e/utils/manager.go** | Uses `FlintlockHosts` to configure CAPMVM provider; creates workload clusters that call Flintlock gRPC. |
| **test/e2e/config/e2e_conf_v1beta2.yaml** | E2e config (CAPI v1beta2, images, intervals). No direct host config; cluster templates reference root/kernel images. |
| **test/e2e/data/infrastructure-microvm/v1alpha1/cluster-template.yaml** (and v1alpha2) | Cluster templates: root image `capmvm-kubernetes:1.23.10`, kernel `flintlock-kernel:5.10.77`, network (macvtap/tap). VIP and network must match host (e.g. 10.42.0.2 for eno4 NAT subnet). |

### 1.3 External / not in repo

- **Flintlock**: installed on host (e.g. `/usr/local/bin/flintlockd`), systemd unit `flintlockd.service`.
- **Flintlock provision script**: `https://github.com/liquidmetal-dev/flintlock` → `hack/scripts/provision.sh devpool` (run on host to create thinpool).
- **containerd**: host package (e.g. `/usr/bin/containerd`); main instance may be used by Docker; we add **containerd-dev** for devmapper.
- **Firecracker**: host binary (e.g. Liquid Metal fork with macvtap); must be in PATH.
- **dmsetup, bc**: host packages for thinpool provisioning.

---

## 2. Troubleshooting steps used (chronological)

1. **Identify failure**  
   - E2e or CAPMVM controller: MicroVMs stuck in "Provisioning".  
   - On host: `journalctl -u flintlockd` showed `snapshotter not loaded: devmapper: invalid argument` during `runtime_volume_mount`.

2. **Document devmapper in README**  
   - Added "snapshotter not loaded: devmapper" bullet and link to Flintlock containerd docs.  
   - Added full **Connect and setup devmapper** section (SSH, deps, provision devpool, containerd-dev config, start containerd-dev, Flintlock drop-in, pre-pull images, re-run e2e).

3. **Clarify dmsetup**  
   - User ran `dmsetup ls` without sudo and saw "Permission denied" and "Incompatible libdevmapper".  
   - README updated to use `sudo dmsetup ls` and to note that the message is from lack of root, not a failed thinpool.

4. **Automate host update**  
   - Added `hack/containerd-config-dev.toml` and `hack/containerd-dev.service`.  
   - SCP + SSH to host (root@192.168.1.57): install config and service, enable/start containerd-dev, add flintlockd drop-in with `--containerd-socket /run/containerd-dev/containerd.sock` and `--parent-iface eno4`, restart flintlockd.  
   - Pre-pull images into containerd-dev `flintlock` namespace (capmvm-kubernetes:1.23.10, flintlock-kernel:5.10.77).

5. **Connect and troubleshoot via SSH**  
   - SSH as casibbald@192.168.1.57: verified containerd-dev and flintlockd active, socket and port 9090, grpcurl list and ListMicroVMs, KVM and Firecracker.  
   - Could not run sudo (no TTY/password): thinpool and ctr images checks skipped in that session.  
   - Noted **eno4 DOWN**: Flintlock was configured with `--parent-iface eno4` but the wired interface had no link (no switch).  
   - Conclusion: network switch required to bring eno4 up and get L2 for MicroVMs.

6. **Commit and push**  
   - Committed README + hack assets with `farm git commit` / `farm git push` (conventional message).

---

## 3. Host aspects to check (for healthcheck and Ansible)

| Aspect | Check | Required for |
|--------|--------|----------------|
| **containerd-dev** | Service active; socket exists at `/run/containerd-dev/containerd.sock`. | Flintlock to mount root volumes (devmapper). |
| **Thinpool** | `dmsetup ls` shows `flintlock-dev-thinpool`. | Devmapper snapshotter. |
| **Containerd-dev config** | `/etc/containerd/config-dev.toml` with devmapper plugin and correct pool_name. | Correct snapshotter and paths. |
| **Flintlock** | Service active; listening on `0.0.0.0:9090`. | CAPMVM and e2e to call CreateMicroVM/ListMicroVMs. |
| **Flintlock → containerd-dev** | Flintlock ExecStart includes `--containerd-socket /run/containerd-dev/containerd.sock`. | Flintlock uses devmapper-backed containerd. |
| **Images** | In containerd-dev, namespace `flintlock`: root image (e.g. `capmvm-kubernetes:1.23.10`) and kernel (e.g. `flintlock-kernel:5.10.77`). | CreateMicroVM can mount root and kernel. |
| **KVM** | `/dev/kvm` exists and is readable by flintlock (e.g. kvm group). | Firecracker to run MicroVMs. |
| **Firecracker** | Binary in PATH, `firecracker --version` succeeds. | MicroVM runtime. |
| **Kernel modules** | `macvlan` (and optionally `macvtap`) loaded. | Macvtap networking for MicroVMs. |
| **Parent interface** | Interface passed to `--parent-iface` exists and is UP (e.g. `ip link show eno4`). | Macvtap devices can be created. |
| **gRPC API** | `grpcurl -plaintext 127.0.0.1:9090 list` shows `microvm.services.api.v1alpha1.MicroVM`. | E2e and CAPMVM can talk to Flintlock. |
| **Optional: ListMicroVMs** | `grpcurl -plaintext -d '{}' 127.0.0.1:9090 .../ListMicroVMs` returns (no transport error). | Full API path works. |

Optional for e2e (depending on where Kind runs): **Docker** running (if Kind is on same host); **IP forwarding / iptables / dnsmasq** if using NAT on the parent interface.

---

## 4. Bootstrap healthcheck script

- **Path**: `test/e2e/bootstrap-healthcheck.py`
- **Purpose**: Run on the Flintlock host (or via SSH) to verify all aspects above in one go.
- **Usage**:
  - On host as root: `sudo python3 test/e2e/bootstrap-healthcheck.py` (or `./bootstrap-healthcheck.py`) for full checks including thinpool and images.
  - Remote (e.g. from e2e machine): `python3 bootstrap-healthcheck.py --host 192.168.1.57` for reachability and gRPC only (no sudo on host).
- **Output**: Pass/fail per check; optional `--json` for machine-readable output. Exit code 0 if all required checks pass.
- **Implementation**: Python 3; subprocess for system commands; no shell scripts. See script docstring and `--help`.

---

## 5. Ansible playbook (new host setup)

- **Path**: `hack/ansible/`
- **Purpose**: Idempotent setup of a new Linux host as a Flintlock/containerd-dev host for CAPMVM e2e.
- **Contents** (see directory):
  - **playbook.yml**: Main playbook (target: `flintlock_hosts`).
  - **vars.yml**: Parent interface name, Flintlock gRPC port, root/kernel image refs, socket paths, thinpool name.
  - **inventory.example.yml**: Example inventory; copy to `inventory.yml` and set your host(s).
  - **README.md**: Usage, prerequisites, and post-playbook steps.
- **Tasks**: Install dmsetup/bc; create dirs; clone Flintlock and run `provision.sh devpool`; deploy containerd-config-dev.toml and containerd-dev.service; enable/start containerd-dev; create flintlockd drop-in; restart flintlockd; pre-pull root and kernel images.
- **Usage**: From repo root, `ansible-playbook -i inventory.yml hack/ansible/playbook.yml`. Override parent interface with `-e flintlock_parent_iface=eth0`.
- **Note**: Firecracker and Flintlock binary install are not in this playbook; the host must have them. Optional network (NAT, dnsmasq) can be added as extra tasks.

---

## 6. Summary

- **Code touch points**: README (troubleshooting + devmapper), hack/containerd-config-dev.toml, hack/containerd-dev.service, e2e params/defaults/templates. **New**: BOOTSTRAP_AUDIT.md, bootstrap-healthcheck.py, hack/ansible/.
- **Troubleshooting**: Devmapper error → document and run devpool + containerd-dev + Flintlock drop-in + images; dmsetup sudo note; eno4 DOWN → need network switch for full e2e.
- **Automation**: **bootstrap-healthcheck.py** checks all host aspects; **Ansible playbook** sets up a new Linux host up to the point of “ready for e2e” (excluding physical network/switch).
