# Ansible: Flintlock/containerd-dev host setup

This directory contains an Ansible playbook to set up a Linux host as a Flintlock host with containerd-dev (devmapper) for CAPMVM e2e.

## Prerequisites on the host

- **flintlockd** and **Firecracker** in PATH (e.g. from [Flintlock provision script](https://github.com/liquidmetal-dev/flintlock/blob/main/hack/scripts/provision.sh) or [Flintlock docs](https://flintlock.liquidmetal.dev/docs/getting-started/containerd/)) — the playbook does not install these.
- **KVM** available (`/dev/kvm`)
- Ansible controller can SSH as a user with sudo

When `install_containerd` is true (default), the playbook installs **containerd** via apt so the host has `/usr/bin/containerd` and `ctr`. When `install_docker` is true, it installs **Docker** (docker.io) for Kind when e2e runs on the same host.

## What the playbook does

1. Installs apt packages: **thin-provisioning-tools**, **lvm2**, **git**, **curl**, **wget**, **dmsetup**, **bc** (same set as Flintlock’s `provision.sh` apt step); optionally **containerd** and **docker.io**
2. Creates containerd-dev directories
3. Clones Flintlock repo and runs `provision.sh devpool` (creates thinpool)
4. Deploys `hack/containerd-config-dev.toml` to `/etc/containerd/config-dev.toml`
5. Deploys `hack/containerd-dev.service` to `/etc/systemd/system/`
6. Enables and starts `containerd-dev`
7. Creates flintlockd drop-in so Flintlock uses containerd-dev socket and either `--parent-iface` (macvtap) or `--bridge-name` (tap), depending on `flintlock_network_mode`
8. Restarts flintlockd
9. Pulls images into the `flintlock` namespace with `ctr` (default: root and kernel images; configurable via `flintlock_ctr_images`)

## Usage

From the **repo root**:

```bash
# Example inventory: single host
echo "192.168.1.57" > /tmp/inv
echo "[flintlock_hosts]" >> /tmp/inv
echo "192.168.1.57" >> /tmp/inv

ansible-playbook -i /tmp/inv hack/ansible/playbook.yml
```

Override network mode (tap vs macvtap) and interfaces:

```bash
# macvtap (default): use parent interface
ansible-playbook -i /tmp/inv hack/ansible/playbook.yml -e "flintlock_network_mode=macvtap" -e "flintlock_parent_iface=eth0"

# tap: use bridge (must exist on host)
ansible-playbook -i /tmp/inv hack/ansible/playbook.yml -e "flintlock_network_mode=tap" -e "flintlock_bridge_name=br-mvm"
```

## After the playbook

1. **Network**: If the parent interface (e.g. eno4) is used for macvtap, ensure it is up and connected (e.g. to a switch). See `test/e2e/README.md` (Troubleshooting the Linux host, NAT optional).
2. **Healthcheck**: From the repo, run the bootstrap healthcheck on the host:
   ```bash
   ssh root@192.168.1.57 'cd /path/to/repo && sudo python3 test/e2e/bootstrap-healthcheck.py --parent-iface eno4'
   ```
   Or from your machine (remote checks only): `python3 test/e2e/bootstrap-healthcheck.py --host 192.168.1.57`
3. **E2e**: Run e2e with `-e2e.flintlock-hosts=192.168.1.57:9090` and appropriate `-e2e.capmvm.vip-address`.

## Variables

See `vars.yml`. Important overrides:

- **`flintlock_network_mode`**: `macvtap` or `tap` (must match cluster template `networkInterfaces[].type`). Default `macvtap`.
- **`flintlock_parent_iface`**: parent interface for macvtap (e.g. `eno4`, `eth0`). Used when `flintlock_network_mode=macvtap`.
- **`flintlock_bridge_name`**: bridge for tap (e.g. `br-mvm`). Used when `flintlock_network_mode=tap`; bridge must exist on the host.
- **`flintlock_ctr_images`**: list of image refs to pull with `ctr` into the `flintlock` namespace (default: root and kernel images). Add more entries if your templates need them.
- **`flintlock_root_image`**, **`flintlock_kernel_image`**: default image refs; included in `flintlock_ctr_images` unless overridden.
- **`install_containerd`**: when true (default), install containerd via apt. Set false if the host already has containerd.
- **`install_docker`**: when true, install Docker (docker.io) and start the service — for when Kind/e2e runs on the same host.
- **`apt_packages_base`**: list of base packages (thin-provisioning-tools, lvm2, git, curl, wget, dmsetup, bc); extend or override as needed.
