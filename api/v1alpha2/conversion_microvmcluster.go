// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1 "github.com/liquidmetal-dev/cluster-api-provider-microvm/api/v1alpha1"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	clusterv1beta2 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

// ConvertTo converts this MicrovmCluster to the Hub version (v1alpha1).
func (m *MicrovmCluster) ConvertTo(dst conversion.Hub) error {
	hub := dst.(*infrav1.MicrovmCluster)
	hub.ObjectMeta = m.ObjectMeta
	// Spec
	hub.Spec.ControlPlaneEndpoint = clusterv1beta1.APIEndpoint{
		Host: m.Spec.ControlPlaneEndpoint.Host,
		Port: m.Spec.ControlPlaneEndpoint.Port,
	}
	hub.Spec.SSHPublicKeys = m.Spec.SSHPublicKeys
	hub.Spec.Placement = convertPlacementToV1Alpha1(m.Spec.Placement)
	hub.Spec.MicrovmProxy = m.Spec.MicrovmProxy
	hub.Spec.TLSSecretRef = m.Spec.TLSSecretRef
	// Status
	hub.Status.Ready = m.Status.Ready
	hub.Status.Conditions = m.Status.Conditions
	hub.Status.FailureDomains = failureDomainsSliceToMap(m.Status.FailureDomains)
	return nil
}

// ConvertFrom converts from the Hub version (v1alpha1) to this version.
func (m *MicrovmCluster) ConvertFrom(src conversion.Hub) error {
	hub := src.(*infrav1.MicrovmCluster)
	m.ObjectMeta = hub.ObjectMeta
	// Spec
	m.Spec.ControlPlaneEndpoint = clusterv1beta2.APIEndpoint{
		Host: hub.Spec.ControlPlaneEndpoint.Host,
		Port: hub.Spec.ControlPlaneEndpoint.Port,
	}
	m.Spec.SSHPublicKeys = hub.Spec.SSHPublicKeys
	m.Spec.Placement = convertPlacementFromV1Alpha1(hub.Spec.Placement)
	m.Spec.MicrovmProxy = hub.Spec.MicrovmProxy
	m.Spec.TLSSecretRef = hub.Spec.TLSSecretRef
	// Status
	m.Status.Ready = hub.Status.Ready
	m.Status.Conditions = hub.Status.Conditions
	m.Status.FailureDomains = failureDomainsMapToSlice(hub.Status.FailureDomains)
	return nil
}

func convertPlacementToV1Alpha1(p Placement) infrav1.Placement {
	out := infrav1.Placement{}
	if p.StaticPool != nil {
		out.StaticPool = &infrav1.StaticPoolPlacement{
			BasicAuthSecret: p.StaticPool.BasicAuthSecret,
			Hosts:          make([]infrav1.MicrovmHost, len(p.StaticPool.Hosts)),
		}
		for i := range p.StaticPool.Hosts {
			out.StaticPool.Hosts[i] = infrav1.MicrovmHost{
				Name:                p.StaticPool.Hosts[i].Name,
				Endpoint:            p.StaticPool.Hosts[i].Endpoint,
				ControlPlaneAllowed: p.StaticPool.Hosts[i].ControlPlaneAllowed,
			}
		}
	}
	return out
}

func convertPlacementFromV1Alpha1(p infrav1.Placement) Placement {
	out := Placement{}
	if p.StaticPool != nil {
		out.StaticPool = &StaticPoolPlacement{
			BasicAuthSecret: p.StaticPool.BasicAuthSecret,
			Hosts:          make([]MicrovmHost, len(p.StaticPool.Hosts)),
		}
		for i := range p.StaticPool.Hosts {
			out.StaticPool.Hosts[i] = MicrovmHost{
				Name:                p.StaticPool.Hosts[i].Name,
				Endpoint:            p.StaticPool.Hosts[i].Endpoint,
				ControlPlaneAllowed: p.StaticPool.Hosts[i].ControlPlaneAllowed,
			}
		}
	}
	return out
}

func failureDomainsSliceToMap(s []clusterv1beta2.FailureDomain) clusterv1beta1.FailureDomains {
	if s == nil {
		return nil
	}
	out := make(clusterv1beta1.FailureDomains, len(s))
	for i := range s {
		fd := &s[i]
		spec := clusterv1beta1.FailureDomainSpec{}
		if fd.ControlPlane != nil {
			spec.ControlPlane = *fd.ControlPlane
		}
		spec.Attributes = fd.Attributes
		out[fd.Name] = spec
	}
	return out
}

func failureDomainsMapToSlice(m clusterv1beta1.FailureDomains) []clusterv1beta2.FailureDomain {
	if m == nil {
		return nil
	}
	out := make([]clusterv1beta2.FailureDomain, 0, len(m))
	for name, spec := range m {
		cp := spec.ControlPlane
		out = append(out, clusterv1beta2.FailureDomain{
			Name:         name,
			ControlPlane: &cp,
			Attributes:   spec.Attributes,
		})
	}
	return out
}
