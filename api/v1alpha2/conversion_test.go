// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	clusterv1beta2 "sigs.k8s.io/cluster-api/api/core/v1beta2"

	infrav1 "github.com/liquidmetal-dev/cluster-api-provider-microvm/api/v1alpha1"
)

func TestMicrovmCluster_ConvertTo_ConvertFrom_RoundTrip(t *testing.T) {
	g := NewWithT(t)

	// Build v1alpha2 cluster with placement and failure domains
	cpTrue := true
	m := &MicrovmCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cluster", Namespace: "default"},
		Spec: MicrovmClusterSpec{
			ControlPlaneEndpoint: clusterv1beta2.APIEndpoint{Host: "1.2.3.4", Port: 6443},
			Placement: Placement{
				StaticPool: &StaticPoolPlacement{
					Hosts: []MicrovmHost{
						{Name: "host1", Endpoint: "127.0.0.1:9090", ControlPlaneAllowed: true},
					},
					BasicAuthSecret: "secret",
				},
			},
		},
		Status: MicrovmClusterStatus{
			Ready: true,
			FailureDomains: []clusterv1beta2.FailureDomain{
				{Name: "fd1", ControlPlane: &cpTrue, Attributes: map[string]string{"endpoint": "a:9090"}},
			},
		},
	}

	hub := &infrav1.MicrovmCluster{}
	g.Expect(m.ConvertTo(hub)).To(Succeed())

	g.Expect(hub.Name).To(Equal("test-cluster"))
	g.Expect(hub.Spec.ControlPlaneEndpoint.Host).To(Equal("1.2.3.4"))
	g.Expect(hub.Spec.Placement.StaticPool).NotTo(BeNil())
	g.Expect(hub.Spec.Placement.StaticPool.Hosts).To(HaveLen(1))
	g.Expect(hub.Spec.Placement.StaticPool.BasicAuthSecret).To(Equal("secret"))
	g.Expect(hub.Status.Ready).To(BeTrue())
	g.Expect(hub.Status.FailureDomains).To(HaveKey("fd1"))
	g.Expect(hub.Status.FailureDomains["fd1"].ControlPlane).To(BeTrue())
	g.Expect(hub.Status.FailureDomains["fd1"].Attributes).To(HaveKeyWithValue("endpoint", "a:9090"))

	// Convert back to v1alpha2
	m2 := &MicrovmCluster{}
	g.Expect(m2.ConvertFrom(hub)).To(Succeed())
	g.Expect(m2.Name).To(Equal(m.Name))
	g.Expect(m2.Spec.ControlPlaneEndpoint.Host).To(Equal(m.Spec.ControlPlaneEndpoint.Host))
	g.Expect(m2.Spec.Placement.StaticPool).NotTo(BeNil())
	g.Expect(m2.Spec.Placement.StaticPool.Hosts).To(HaveLen(1))
	g.Expect(m2.Status.Ready).To(Equal(m.Status.Ready))
	g.Expect(m2.Status.FailureDomains).To(HaveLen(1))
	g.Expect(m2.Status.FailureDomains[0].Name).To(Equal("fd1"))
}

func TestMicrovmCluster_ConvertFrom_ConvertTo_WithNilPlacementAndFailureDomains(t *testing.T) {
	g := NewWithT(t)

	hub := &infrav1.MicrovmCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "minimal", Namespace: "default"},
		Spec: infrav1.MicrovmClusterSpec{
			Placement: infrav1.Placement{}, // nil StaticPool
		},
		Status: infrav1.MicrovmClusterStatus{
			FailureDomains: nil,
		},
	}

	m := &MicrovmCluster{}
	g.Expect(m.ConvertFrom(hub)).To(Succeed())
	g.Expect(m.Spec.Placement.StaticPool).To(BeNil())
	g.Expect(m.Status.FailureDomains).To(BeNil())

	hub2 := &infrav1.MicrovmCluster{}
	g.Expect(m.ConvertTo(hub2)).To(Succeed())
	g.Expect(hub2.Spec.Placement.StaticPool).To(BeNil())
	g.Expect(hub2.Status.FailureDomains).To(BeNil())
}

func TestMicrovmCluster_FailureDomainsMapToSliceAndBack(t *testing.T) {
	g := NewWithT(t)

	// Hub (v1alpha1) uses map
	hub := &infrav1.MicrovmCluster{
		Status: infrav1.MicrovmClusterStatus{
			FailureDomains: clusterv1beta1.FailureDomains{
				"fd1": clusterv1beta1.FailureDomainSpec{ControlPlane: true, Attributes: map[string]string{"k": "v"}},
			},
		},
	}
	m := &MicrovmCluster{}
	g.Expect(m.ConvertFrom(hub)).To(Succeed())
	g.Expect(m.Status.FailureDomains).To(HaveLen(1))
	g.Expect(m.Status.FailureDomains[0].Name).To(Equal("fd1"))

	hub2 := &infrav1.MicrovmCluster{}
	g.Expect(m.ConvertTo(hub2)).To(Succeed())
	g.Expect(hub2.Status.FailureDomains).To(HaveKey("fd1"))
	g.Expect(hub2.Status.FailureDomains["fd1"].ControlPlane).To(BeTrue())
}

func TestMicrovmMachine_ConvertTo_ConvertFrom_RoundTrip(t *testing.T) {
	g := NewWithT(t)

	providerID := "microvm://fd1/abc"
	m := &MicrovmMachine{
		ObjectMeta: metav1.ObjectMeta{Name: "machine-1", Namespace: "default"},
		Spec: MicrovmMachineSpec{
			ProviderID: ptr.To(providerID),
		},
		Status: MicrovmMachineStatus{
			Ready:   true,
			VMState: nil,
			Addresses: []clusterv1beta2.MachineAddress{
				{Type: clusterv1beta2.MachineInternalIP, Address: "10.0.0.1"},
			},
		},
	}

	hub := &infrav1.MicrovmMachine{}
	g.Expect(m.ConvertTo(hub)).To(Succeed())

	g.Expect(hub.Spec.ProviderID).NotTo(BeNil())
	g.Expect(*hub.Spec.ProviderID).To(Equal(providerID))
	g.Expect(hub.Status.Ready).To(BeTrue())
	g.Expect(hub.Status.Addresses).To(HaveLen(1))
	g.Expect(hub.Status.Addresses[0].Type).To(Equal(clusterv1beta1.MachineInternalIP))
	g.Expect(hub.Status.Addresses[0].Address).To(Equal("10.0.0.1"))

	m2 := &MicrovmMachine{}
	g.Expect(m2.ConvertFrom(hub)).To(Succeed())
	g.Expect(m2.Spec.ProviderID).NotTo(BeNil())
	g.Expect(*m2.Spec.ProviderID).To(Equal(providerID))
	g.Expect(m2.Status.Addresses).To(HaveLen(1))
	g.Expect(m2.Status.Addresses[0].Address).To(Equal("10.0.0.1"))
}

func TestMicrovmMachine_ConvertWithNilAddresses(t *testing.T) {
	g := NewWithT(t)

	hub := &infrav1.MicrovmMachine{
		ObjectMeta: metav1.ObjectMeta{Name: "m", Namespace: "default"},
		Status: infrav1.MicrovmMachineStatus{
			Addresses: nil,
		},
	}
	m := &MicrovmMachine{}
	g.Expect(m.ConvertFrom(hub)).To(Succeed())
	g.Expect(m.Status.Addresses).To(BeNil())

	hub2 := &infrav1.MicrovmMachine{}
	g.Expect(m.ConvertTo(hub2)).To(Succeed())
	g.Expect(hub2.Status.Addresses).To(BeNil())
}

func TestMicrovmMachineTemplate_ConvertTo_ConvertFrom_RoundTrip(t *testing.T) {
	g := NewWithT(t)

	m := &MicrovmMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "template-1", Namespace: "default"},
		Spec: MicrovmMachineTemplateSpec{
			Template: MicrovmMachineTemplateResource{
				ObjectMeta: clusterv1beta2.ObjectMeta{
					Labels:      map[string]string{"label": "value"},
					Annotations: map[string]string{"ann": "value"},
				},
				Spec: MicrovmMachineSpec{
					ProviderID: ptr.To("microvm://fd1/xyz"),
				},
			},
		},
	}

	hub := &infrav1.MicrovmMachineTemplate{}
	g.Expect(m.ConvertTo(hub)).To(Succeed())

	g.Expect(hub.Spec.Template.ObjectMeta.Labels).To(HaveKeyWithValue("label", "value"))
	g.Expect(hub.Spec.Template.ObjectMeta.Annotations).To(HaveKeyWithValue("ann", "value"))
	g.Expect(hub.Spec.Template.Spec.ProviderID).NotTo(BeNil())

	m2 := &MicrovmMachineTemplate{}
	g.Expect(m2.ConvertFrom(hub)).To(Succeed())
	g.Expect(m2.Spec.Template.ObjectMeta.Labels).To(HaveKeyWithValue("label", "value"))
	g.Expect(m2.Spec.Template.Spec.ProviderID).NotTo(BeNil())
}
