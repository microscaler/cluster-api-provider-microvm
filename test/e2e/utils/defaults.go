//go:build e2e
// +build e2e

package utils

import (
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/runtime"
	cgscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/cluster-api/test/framework"

	infrav1 "github.com/liquidmetal-dev/cluster-api-provider-microvm/api/v1alpha1"
	infrav1alpha2 "github.com/liquidmetal-dev/cluster-api-provider-microvm/api/v1alpha2"
)

const (
	// DefaultE2EConfig is the default location for the E2E config file.
	// Must use v1beta2 to match CAPI test framework (clusterctl v1.11.x only supports v1beta2 management clusters).
	DefaultE2EConfig = "config/e2e_conf_v1beta2.yaml"
	// DefaultKubernetesVersion is the default version of Kubernetes which will
	// the workload cluster will run.
	// DefaultKubernetesVersion must be >= 1.22.0: CAPI kubeadm-bootstrap v1.11+ does not support older versions.
	// Use latest Liquid Metal release: ghcr.io/liquidmetal-dev/capmvm-kubernetes:1.23.10
	DefaultKubernetesVersion = "1.23.10"
	// DefaultVIPAddress is the default address which the workload cluster's
	// load balancer will use. When e2e runs on the same host as Flintlock with
	// Kind, override via -e2e.capmvm.vip-address with the Kind network gateway
	// so the CAPI controller in Kind can reach the workload API.
	DefaultVIPAddress = "192.168.1.25"

	DefaultSkipCleanup     = false
	DefaultExistingCluster = false
)

// Flavour consts for clusterctl template selection.
const (
	Vanilla       = ""
	Cilium        = "cilium"
	V1Alpha2      = "v1alpha2"
	V1Alpha2Cilium = "v1alpha2-cilium"
)

// DefaultScheme returns the default scheme to use for testing.
func DefaultScheme() *runtime.Scheme {
	sc := runtime.NewScheme()
	framework.TryAddDefaultSchemes(sc)
	_ = infrav1.AddToScheme(sc)
	_ = infrav1alpha2.AddToScheme(sc)
	_ = cgscheme.AddToScheme(sc)

	return sc
}

func DefaultArtefactDir() string {
	pwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	return filepath.Join(pwd, "_artefacts")
}
