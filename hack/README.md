# Hack: host setup and troubleshooting

This directory holds configs and automation for setting up a Linux host as a Flintlock/containerd host for CAPMVM e2e, and troubleshooting steps when tests or real clusters use a remote Flintlock host.

**Contents**

- **containerd-config-dev.toml** – Containerd config with devmapper snapshotter for Flintlock. Deploy to `/etc/containerd/config-dev.toml` on the host.
- **containerd-dev.service** – Systemd unit for containerd-dev. Deploy to `/etc/systemd/system/containerd-dev.service`.
- **ansible/** – Playbook and vars to set up a new Linux host (thinpool, containerd-dev, Flintlock drop-in, pre-pull images). See [ansible/README.md](ansible/README.md).

**Automation**

- **Bootstrap healthcheck**: `test/e2e/bootstrap-healthcheck.py` – run on the Flintlock host (as root) or with `--host <ip>` for remote checks. See [test/e2e/BOOTSTRAP_AUDIT.md](../test/e2e/BOOTSTRAP_AUDIT.md).
- **Ansible**: from repo root, `ansible-playbook -i inventory hack/ansible/playbook.yml`. See [ansible/README.md](ansible/README.md).

---

## Network interface options: tap and macvtap

MicroVM network interfaces in the cluster template (`networkInterfaces[].type`) can be **`tap`** or **`macvtap`**. The host and Firecracker setup depends on which you use.

| Option | Template value | Host requirement | Flintlock flag | Firecracker |
|--------|----------------|------------------|----------------|-------------|
| **tap** | `type: "tap"` | A **bridge** with an IP and DHCP (or existing LAN). Create e.g. `br-mvm`, attach the host interface, assign an IP, run DHCP on the bridge or subnet. | `--bridge-name <bridge>` (e.g. `--bridge-name br-mvm`) | **Upstream** Firecracker supports tap. No custom build. |
| **macvtap** | `type: "macvtap"` | A **parent interface** (physical or bond) that has L2 connectivity to the network you want for MicroVMs. No bridge required. | `--parent-iface <interface>` (e.g. `--parent-iface eth0` or `eno4`) | **Liquid Metal’s Firecracker fork** with macvtap support (e.g. [liquidmetal-dev/firecracker](https://github.com/liquidmetal-dev/firecracker), `feature/macvtap` branch). Upstream Firecracker does **not** support macvtap. |

- **Which template uses which?**  
  - Repo root **templates/** (e.g. `templates/cluster-template.yaml`) use **macvtap** by default.  
  - **test/e2e/data/** cluster templates use **tap** by default (works with upstream Firecracker; e2e host needs a bridge and `--bridge-name`).  
  You can change the template: set `type: "tap"` or `type: "macvtap"` on each `networkInterfaces` entry to match your host and Firecracker.

- **Summary**: Use **tap** if you have a bridge and upstream Firecracker; use **macvtap** if you have a parent interface and the Liquid Metal Firecracker fork.

---

## Troubleshooting the Linux host

When e2e or a real cluster uses a remote Flintlock host, run these checks on that host (as root where noted).

### Setting up the Linux host for macvtap

If your cluster template uses **macvtap** (e.g. repo `templates/`), the Flintlock host must have a suitable **parent interface** for macvtap (the physical or bond interface that will carry macvlan/macvtap traffic). No bridge is required for macvtap.

1. **Choose a parent interface**  
   Pick an interface that has L2 connectivity to the network you want MicroVMs to use (e.g. the same LAN as the management cluster or DHCP). Examples: `eth0`, `ens18`, or a bond. Check with `ip -br addr show`.

2. **Ensure kernel support**  
   Macvtap uses the macvlan driver. Check: `lsmod | grep macvlan`. If missing, load with `modprobe macvlan` (and `modprobe macvtap` if your distro has it as a separate module).

3. **Configure Flintlock to use the parent interface**  
   Set the parent interface so the microvm provider can create macvtap devices. Either:
   - **CLI**: `flintlockd run --grpc-endpoint 0.0.0.0:9090 --parent-iface eth0` (replace `eth0` with your interface).
   - **Config file**: In `/etc/opt/flintlockd/config.yaml` set `parent-iface: eth0` (see [Flintlock network docs](https://flintlock.liquidmetal.dev/docs/getting-started/network)).
   - **Systemd**: Add `--parent-iface <interface>` to `ExecStart` in the service or a drop-in under `/etc/systemd/system/flintlockd.service.d/`, then `systemctl daemon-reload && systemctl restart flintlockd`.

4. **Verify**  
   After creating a MicroVM, on the host run `ip link show type macvlan` (or `ip link | grep macvtap`) to see child interfaces. If you still see "macvtap network interfaces not supported", **Firecracker** (the microvm provider) likely does not support macvtap: use Liquid Metal’s Firecracker fork from the `feature/macvtap` branch.

5. **Wired interface without uplink (NAT via WiFi)**  
   If you have no cable in the wired port but want to use it as the macvtap parent (e.g. WiFi doesn’t support macvlan well), you can bring up the wired interface with a private subnet and NAT traffic to the internet via your WiFi interface. On the host (as root): (a) Assign an IP to the wired interface, e.g. `ip addr add 10.42.0.1/24 dev eno4` and `ip link set eno4 up`. (b) Enable forwarding: `sysctl -w net.ipv4.ip_forward=1` (and persist in `/etc/sysctl.conf`). (c) iptables: `iptables -t nat -A POSTROUTING -s 10.42.0.0/24 ! -d 10.42.0.0/24 -o wlo5 -j MASQUERADE` and `iptables -A FORWARD -i eno4 -o wlo5 -j ACCEPT` plus `iptables -A FORWARD -i wlo5 -o eno4 -m state --state RELATED,ESTABLISHED -j ACCEPT` (replace `eno4`/`wlo5` with your wired/WiFi interface). Persist with `iptables-save > /etc/iptables/rules.v4` if using iptables-persistent. (d) Run a DHCP server on the wired interface so MicroVMs get IPs in 10.42.0.0/24 (e.g. dnsmasq with `interface=eno4`, `dhcp-range=10.42.0.10,10.42.0.200,255.255.255.0`, gateway `10.42.0.1`). (e) Configure Flintlock with `--parent-iface eno4`. For e2e, set the workload cluster VIP in that subnet, e.g. `E2E_CAPMVM_VIP=10.42.0.2` (use an IP outside the DHCP range, e.g. reserve .2 for the control plane VIP).

### Setting up the Linux host for tap

If your cluster template uses **tap** (e.g. `test/e2e/data/` cluster templates), the Flintlock host needs a **bridge** that Firecracker can attach tap devices to.

1. **Create a bridge** (e.g. `br-mvm`) and give it an IP in the subnet you want for MicroVMs. Attach the host interface that has (or will have) L2 connectivity to that network.
2. **DHCP**: Run a DHCP server on the bridge (or that subnet) so MicroVMs get IPs (e.g. dnsmasq with `interface=br-mvm` and a `dhcp-range`).
3. **Configure Flintlock** with `--bridge-name br-mvm` (or your bridge name). No `--parent-iface` is required for tap.
4. **Firecracker**: Upstream Firecracker supports tap; no Liquid Metal fork needed.

See [Flintlock network docs](https://flintlock.liquidmetal.dev/docs/getting-started/network) for more detail.

### containerd

- Flintlock talks to containerd for kernel/rootfs/images. By default it uses **`/run/containerd/containerd.sock`** (main containerd). If you use a separate **containerd-dev** (e.g. Flintlock provision script with `--dev`), you must pass the dev socket to flintlock (see below).
- Check: `systemctl is-active containerd` and optionally `containerd-dev`. Sockets: `ls -la /run/containerd/containerd.sock /run/containerd-dev/containerd.sock`.
- **Critical**: The containerd instance Flintlock uses must have the **images** (kernel, rootfs, etc.) in the **`flintlock`** namespace. List with:
  ```bash
  ctr -a /run/containerd/containerd.sock -n flintlock images ls
  ```
  If this is empty, CreateMicroVM will fail when CAPMVM requests a machine. Pre-pull the images your cluster template uses into that namespace (see **Pre-pulling images** below), or use a containerd that already has them (e.g. containerd-dev if you push there).

### Pre-pulling images into the flintlock namespace

On the Linux host where Flintlock runs, pull the root and kernel images into the **same containerd socket and namespace** that Flintlock uses. The e2e cluster templates use these by default:

| Variable | Default image |
|----------|----------------|
| `MVM_ROOT_IMAGE` | `ghcr.io/liquidmetal-dev/capmvm-kubernetes:1.23.10` |
| `MVM_KERNEL_IMAGE` | `ghcr.io/liquidmetal-dev/flintlock-kernel:5.10.77` |

Run as root (replace the socket path if Flintlock uses containerd-dev, e.g. `/run/containerd-dev/containerd.sock`):

```bash
SOCKET="/run/containerd/containerd.sock"
NS="flintlock"

ctr -a "$SOCKET" -n "$NS" images pull ghcr.io/liquidmetal-dev/capmvm-kubernetes:1.23.10
ctr -a "$SOCKET" -n "$NS" images pull ghcr.io/liquidmetal-dev/flintlock-kernel:5.10.77
```

Verify: `ctr -a "$SOCKET" -n "$NS" images ls`

### flintlock

- Service: `systemctl status flintlockd`. Listen address: `ss -tlnp | grep 9090` (should be **`0.0.0.0:9090`** so any client can reach it).
- If using **containerd-dev**, override the socket in the service. Create a drop-in or override:
  ```bash
  mkdir -p /etc/systemd/system/flintlockd.service.d
  printf '%s\n' '[Service]' 'ExecStart=' '/usr/local/bin/flintlockd run --containerd-socket /run/containerd-dev/containerd.sock --grpc-endpoint 0.0.0.0:9090' > /etc/systemd/system/flintlockd.service.d/containerd-dev.conf
  systemctl daemon-reload && systemctl restart flintlockd
  ```
  Then ensure the **flintlock** namespace in containerd-dev has the required images.
- Use `--bridge-name <bridge>` for **tap** or `--parent-iface <interface>` for **macvtap**; see [Network interface options: tap and macvtap](#network-interface-options-tap-and-macvtap).

### Docker

- Used for Kind (management cluster). Check: `systemctl is-active docker` and `docker info`. If Kind runs on the same host, Docker must be up. `docker0` can be DOWN if no containers use it; custom bridges (e.g. `br-*`) are normal.

### bridge

- Network type is set in the cluster template (`networkInterfaces[].type`: **tap** or **macvtap**). See [Network interface options: tap and macvtap](#network-interface-options-tap-and-macvtap) above: **tap** requires a host bridge and `--bridge-name`; **macvtap** requires a parent interface and `--parent-iface` (no bridge).
- Check interfaces: `ip -br addr show` and `brctl show` (or `bridge link show`).

### Other checks

- **KVM**: Required for Firecracker. Check: `ls -la /dev/kvm`, `lsmod | grep kvm`.
- **Firecracker**: `which firecracker` and `firecracker --version`.
- **Reachability**: From the machine running e2e (or Kind), ensure the flintlock host:port is reachable: `grpcurl -plaintext <host>:9090 list` (should list `microvm.services.api.v1alpha1.MicroVM`). To list MicroVMs on the host: `hammertime list` or `grpcurl -plaintext -d '{}' 127.0.0.1:9090 microvm.services.api.v1alpha1.MicroVM/ListMicroVMs`.
- **MicroVM spec `version`**: The `version` field in `hammertime list` / ListMicroVMs output is a **spec revision** (increments each time Flintlock saves the spec, e.g. after a reconcile). It is not a crash count. High version + repeated "reconciliation failed" in `journalctl -u flintlockd` usually means the same error is recurring (e.g. snapshotter or image mount failure).
- **"snapshotter not loaded: devmapper"**: If flintlockd logs show *"snapshotter not loaded: devmapper: invalid argument"* during `runtime_volume_mount`, containerd is configured to use (or Flintlock expects) the **devmapper** snapshotter, but it is not loaded. See **Connect and setup devmapper** below, or the [Flintlock containerd docs](https://flintlock.liquidmetal.dev/docs/getting-started/containerd/) and [Liquid Metal containerd troubleshooting](https://liquidmetal.dev/docs/troubleshooting/containerd).

### Connect and setup devmapper

If you see the "snapshotter not loaded: devmapper" error, perform these steps on the **Flintlock host**. Replace `<flintlock-host>` with the host IP or hostname (e.g. `192.168.1.57`) and use root or a user with sudo.

1. **Connect to the host**
   ```bash
   ssh root@<flintlock-host>
   ```

2. **Install dependencies for the thinpool**
   ```bash
   sudo apt update
   sudo apt install -y dmsetup bc
   ```

3. **Provision the devmapper thinpool**  
   Clone Flintlock (if you don’t have it) and run the provision script. This creates a loopback-based thinpool `flintlock-dev-thinpool` (e.g. 100G data + 10G metadata).
   ```bash
   git clone --depth 1 https://github.com/liquidmetal-dev/flintlock.git /tmp/flintlock
   cd /tmp/flintlock
   sudo ./hack/scripts/provision.sh devpool
   ```
   Verify: `sudo dmsetup ls` should list the thinpool. (Without `sudo`, `dmsetup ls` may show "Permission denied" and "Incompatible libdevmapper" — that is from lack of root, not a failed thinpool.)

4. **Create containerd-dev config with devmapper**  
   Copy the repo’s config to the host, or write `/etc/containerd/config-dev.toml` with the devmapper plugin (pool_name `flintlock-dev-thinpool`, root_path `/var/lib/containerd-dev/snapshotter/devmapper`). See `hack/containerd-config-dev.toml` in this repo. Create dirs: `sudo mkdir -p /var/lib/containerd-dev/snapshotter/devmapper /run/containerd-dev`.

5. **Start containerd-dev**  
   Copy `hack/containerd-dev.service` to `/etc/systemd/system/containerd-dev.service`, then:
   ```bash
   sudo systemctl daemon-reload && sudo systemctl enable --now containerd-dev
   ```
   Ensure the socket exists: `ls -la /run/containerd-dev/containerd.sock`.

6. **Point Flintlock at containerd-dev and restart**
   ```bash
   sudo mkdir -p /etc/systemd/system/flintlockd.service.d
   printf '%s\n' '[Service]' 'ExecStart=' '/usr/local/bin/flintlockd run --containerd-socket /run/containerd-dev/containerd.sock --grpc-endpoint 0.0.0.0:9090' > /tmp/flintlock-override.conf
   # Add --parent-iface or other args to ExecStart if you already use them
   sudo cp /tmp/flintlock-override.conf /etc/systemd/system/flintlockd.service.d/containerd-dev.conf
   sudo systemctl daemon-reload && sudo systemctl restart flintlockd
   ```

7. **Pre-pull images into the flintlock namespace** (containerd-dev)
   ```bash
   SOCKET="/run/containerd-dev/containerd.sock"
   NS="flintlock"
   sudo ctr -a "$SOCKET" -n "$NS" images pull ghcr.io/liquidmetal-dev/capmvm-kubernetes:1.23.10
   sudo ctr -a "$SOCKET" -n "$NS" images pull ghcr.io/liquidmetal-dev/flintlock-kernel:5.10.77
   sudo ctr -a "$SOCKET" -n "$NS" images ls
   ```

8. **Re-run e2e**  
   From your e2e machine, run the tests again; MicroVMs should progress past "Provisioning" once the devmapper snapshotter is in use and images are present.

### Summary

Ensure Flintlock uses the correct containerd socket (main or containerd-dev) and the **flintlock** namespace has the required root and kernel images; otherwise CreateMicroVM will fail. KVM and Firecracker must be present; gRPC should be listening on **`0.0.0.0:9090`** and reachable from any client.
