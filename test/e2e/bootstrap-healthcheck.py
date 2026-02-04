#!/usr/bin/env python3
"""
Bootstrap healthcheck for a Flintlock/containerd-dev host used by CAPMVM e2e.

Checks: containerd-dev service & socket, flintlockd service & port, devmapper
thinpool, flintlock namespace images, KVM, Firecracker, macvlan module,
parent interface (optional), and Flintlock gRPC API.

Run on the Flintlock host as root for full checks (thinpool, ctr images).
Run with --host <addr> from another machine for remote checks only (TCP + gRPC).

Usage:
  sudo ./bootstrap-healthcheck.py              # full local checks
  ./bootstrap-healthcheck.py --host 192.168.1.57   # remote (no sudo on host)
  ./bootstrap-healthcheck.py --json               # machine-readable output
"""

from __future__ import annotations

import argparse
import json
import os
import shutil
import socket
import subprocess
import sys
from typing import Any


# Defaults matching test/e2e/README.md and cluster templates
DEFAULT_CONTAINERD_SOCKET = "/run/containerd-dev/containerd.sock"
DEFAULT_FLINTLOCK_PORT = 9090
DEFAULT_ROOT_IMAGE = "ghcr.io/liquidmetal-dev/capmvm-kubernetes:1.23.10"
DEFAULT_KERNEL_IMAGE = "ghcr.io/liquidmetal-dev/flintlock-kernel:5.10.77"
FLINTLOCK_NAMESPACE = "flintlock"
THINPOOL_NAME = "flintlock-dev-thinpool"


def run(cmd: list[str], capture: bool = True, env: dict[str, str] | None = None) -> subprocess.CompletedProcess:
    return subprocess.run(
        cmd,
        capture_output=capture,
        text=True,
        timeout=30,
        env={**(env or {}), **os.environ},
    )


def check_containerd_dev_socket(socket_path: str) -> tuple[bool, str]:
    if os.path.exists(socket_path):
        if os.path.isdir(socket_path):
            return False, f"exists but is a directory: {socket_path}"
        return True, "socket exists"
    return False, f"missing: {socket_path}"


def check_systemd_active(unit: str) -> tuple[bool, str]:
    r = run(["systemctl", "is-active", "--quiet", unit])
    if r.returncode == 0:
        return True, "active"
    out = (r.stdout or "").strip() or (r.stderr or "").strip() or "inactive"
    return False, out


def check_port_listen(host: str, port: int) -> tuple[bool, str]:
    try:
        with socket.create_connection((host, port), timeout=5) as s:
            return True, f"{host}:{port} reachable"
    except OSError as e:
        return False, str(e)


def check_dmsetup_thinpool(pool_name: str) -> tuple[bool, str]:
    r = run(["dmsetup", "ls"])
    if r.returncode != 0:
        return False, r.stderr or r.stdout or "dmsetup failed"
    if pool_name in (r.stdout or ""):
        return True, f"thinpool '{pool_name}' present"
    return False, f"thinpool '{pool_name}' not in dmsetup ls (run as root)"


def check_ctr_images(
    socket_path: str,
    namespace: str,
    root_image: str,
    kernel_image: str,
) -> tuple[bool, str]:
    r = run(
        [
            "ctr",
            "-a",
            socket_path,
            "-n",
            namespace,
            "images",
            "ls",
            "-q",
        ]
    )
    if r.returncode != 0:
        return False, r.stderr or r.stdout or "ctr images ls failed"
    out = (r.stdout or "").strip()
    have_root = root_image in out or any(root_image.split(":")[0] in line for line in out.splitlines())
    have_kernel = kernel_image in out or any(kernel_image.split(":")[0] in line for line in out.splitlines())
    if have_root and have_kernel:
        return True, f"root and kernel images present in namespace '{namespace}'"
    missing = []
    if not have_root:
        missing.append(root_image)
    if not have_kernel:
        missing.append(kernel_image)
    return False, f"missing in namespace '{namespace}': {', '.join(missing)}"


def check_kvm() -> tuple[bool, str]:
    path = "/dev/kvm"
    if not os.path.exists(path):
        return False, f"{path} missing"
    if not os.access(path, os.R_OK):
        return False, f"{path} not readable (check kvm group)"
    return True, "/dev/kvm present and readable"


def check_firecracker() -> tuple[bool, str]:
    firecracker = shutil.which("firecracker")
    if not firecracker:
        return False, "firecracker not in PATH"
    r = run([firecracker, "--version"])
    if r.returncode != 0:
        return False, r.stderr or r.stdout or "firecracker --version failed"
    return True, (r.stdout or "").strip() or firecracker


def check_macvlan_module() -> tuple[bool, str]:
    r = run(["lsmod"])
    if r.returncode != 0:
        return False, "lsmod failed"
    if "macvlan" in (r.stdout or ""):
        return True, "macvlan loaded"
    return False, "macvlan module not loaded (modprobe macvlan)"


def check_parent_interface(iface: str | None) -> tuple[bool, str]:
    if not iface:
        return True, "no parent interface configured (skip)"
    r = run(["ip", "link", "show", iface])
    if r.returncode != 0:
        return False, f"interface '{iface}' not found"
    out = r.stdout or ""
    if "state UP" in out or "state UNKNOWN" in out:
        return True, f"interface '{iface}' present"
    return False, f"interface '{iface}' exists but not UP (connect cable or ip link set {iface} up)"


def check_grpcurl_list(host: str, port: int) -> tuple[bool, str]:
    grpcurl = shutil.which("grpcurl")
    if not grpcurl:
        return True, "grpcurl not installed (skip gRPC list)"
    r = run([grpcurl, "-plaintext", f"{host}:{port}", "list"])
    if r.returncode != 0:
        return False, r.stderr or r.stdout or "grpcurl list failed"
    if "microvm.services.api.v1alpha1.MicroVM" in (r.stdout or ""):
        return True, "Flintlock gRPC API listed"
    return False, "MicroVM service not in list"


def main() -> int:
    ap = argparse.ArgumentParser(
        description="Bootstrap healthcheck for Flintlock/containerd-dev host (CAPMVM e2e)."
    )
    ap.add_argument(
        "--host",
        default="127.0.0.1",
        help="Flintlock host (default 127.0.0.1 for local). Use IP for remote checks only.",
    )
    ap.add_argument(
        "--port",
        type=int,
        default=DEFAULT_FLINTLOCK_PORT,
        help=f"Flintlock gRPC port (default {DEFAULT_FLINTLOCK_PORT}).",
    )
    ap.add_argument(
        "--containerd-socket",
        default=DEFAULT_CONTAINERD_SOCKET,
        help=f"Containerd socket (default {DEFAULT_CONTAINERD_SOCKET}).",
    )
    ap.add_argument(
        "--parent-iface",
        default=os.environ.get("FLINTLOCK_PARENT_IFACE", ""),
        help="Parent interface for macvtap (optional).",
    )
    ap.add_argument(
        "--root-image",
        default=DEFAULT_ROOT_IMAGE,
        help=f"Expected root image in flintlock namespace (default {DEFAULT_ROOT_IMAGE}).",
    )
    ap.add_argument(
        "--kernel-image",
        default=DEFAULT_KERNEL_IMAGE,
        help=f"Expected kernel image in flintlock namespace (default {DEFAULT_KERNEL_IMAGE}).",
    )
    ap.add_argument(
        "--json",
        action="store_true",
        help="Output results as JSON.",
    )
    ap.add_argument(
        "--skip-root-only",
        action="store_true",
        help="Skip checks that require root (thinpool, ctr images). Use for remote run.",
    )
    args = ap.parse_args()

    host = args.host
    port = args.port
    socket_path = args.containerd_socket
    parent_iface = (args.parent_iface or "").strip() or None
    skip_root_only = args.skip_root_only or (host != "127.0.0.1")

    results: list[dict[str, Any]] = []

    def add(name: str, ok: bool, msg: str) -> None:
        results.append({"check": name, "ok": ok, "message": msg})

    # --- Local service and socket (only when host is local) ---
    if host == "127.0.0.1":
        add("containerd-dev.service", *check_systemd_active("containerd-dev"))
        add("containerd-dev.socket", *check_containerd_dev_socket(socket_path))
        add("flintlockd.service", *check_systemd_active("flintlockd"))
    else:
        add("containerd-dev.service", True, "skipped (remote)")
        add("containerd-dev.socket", True, "skipped (remote)")
        add("flintlockd.service", True, "skipped (remote)")

    # --- Port reachable ---
    add("flintlock.port", *check_port_listen(host, port))

    # --- Root-only checks (thinpool, ctr images) ---
    if not skip_root_only:
        add("devmapper.thinpool", *check_dmsetup_thinpool(THINPOOL_NAME))
        add(
            "containerd.images",
            *check_ctr_images(
                socket_path,
                FLINTLOCK_NAMESPACE,
                args.root_image,
                args.kernel_image,
            ),
        )
    else:
        add("devmapper.thinpool", True, "skipped (--skip-root-only or remote)")
        add("containerd.images", True, "skipped (--skip-root-only or remote)")

    # --- KVM and Firecracker (local only for Firecracker path) ---
    if host == "127.0.0.1":
        add("kvm", *check_kvm())
        add("firecracker", *check_firecracker())
        add("macvlan", *check_macvlan_module())
        add("parent_interface", *check_parent_interface(parent_iface))
    else:
        add("kvm", True, "skipped (remote)")
        add("firecracker", True, "skipped (remote)")
        add("macvlan", True, "skipped (remote)")
        add("parent_interface", True, "skipped (remote)")

    # --- gRPC API ---
    add("grpcurl.list", *check_grpcurl_list(host, port))

    # --- Output ---
    all_ok = all(r["ok"] for r in results)

    if args.json:
        print(json.dumps({"ok": all_ok, "checks": results}, indent=2))
    else:
        for r in results:
            status = "PASS" if r["ok"] else "FAIL"
            print(f"  [{status}] {r['check']}: {r['message']}")
        print()
        print("Overall:", "PASS" if all_ok else "FAIL")

    return 0 if all_ok else 1


if __name__ == "__main__":
    sys.exit(main())
