// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package webhook_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/liquidmetal-dev/controller-pkg/types/microvm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"

	infrav1alpha2 "github.com/liquidmetal-dev/cluster-api-provider-microvm/api/v1alpha2"
	"github.com/liquidmetal-dev/cluster-api-provider-microvm/internal/webhook"
)

func TestMicrovmClusterV1alpha2_ValidateCreate(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()
	w := &webhook.MicrovmClusterV1alpha2{}

	t.Run("accepts valid cluster with StaticPool placement", func(t *testing.T) {
		cluster := &infrav1alpha2.MicrovmCluster{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: infrav1alpha2.MicrovmClusterSpec{
				Placement: infrav1alpha2.Placement{
					StaticPool: &infrav1alpha2.StaticPoolPlacement{
						Hosts: []infrav1alpha2.MicrovmHost{
							{Name: "host1", Endpoint: "127.0.0.1:9090", ControlPlaneAllowed: true},
						},
					},
				},
			},
		}
		warnings, err := w.ValidateCreate(ctx, cluster)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(warnings).To(BeEmpty())
	})

	t.Run("rejects cluster with nil placement (no StaticPool)", func(t *testing.T) {
		cluster := &infrav1alpha2.MicrovmCluster{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Spec: infrav1alpha2.MicrovmClusterSpec{
				Placement: infrav1alpha2.Placement{},
			},
		}
		warnings, err := w.ValidateCreate(ctx, cluster)
		g.Expect(err).To(HaveOccurred())
		g.Expect(warnings).NotTo(BeEmpty())
	})

	t.Run("rejects wrong type", func(t *testing.T) {
		warnings, err := w.ValidateCreate(ctx, &infrav1alpha2.MicrovmMachine{})
		g.Expect(err).To(HaveOccurred())
		g.Expect(warnings).To(BeEmpty())
	})
}

func TestMicrovmClusterV1alpha2_Default(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()
	w := &webhook.MicrovmClusterV1alpha2{}

	cluster := &infrav1alpha2.MicrovmCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: infrav1alpha2.MicrovmClusterSpec{
			Placement: infrav1alpha2.Placement{
				StaticPool: &infrav1alpha2.StaticPoolPlacement{
					Hosts: []infrav1alpha2.MicrovmHost{{Endpoint: "127.0.0.1:9090"}},
				},
			},
		},
	}
	err := w.Default(ctx, cluster)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestMicrovmClusterV1alpha2_ValidateDelete(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()
	w := &webhook.MicrovmClusterV1alpha2{}

	cluster := &infrav1alpha2.MicrovmCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
	}
	warnings, err := w.ValidateDelete(ctx, cluster)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(warnings).To(BeEmpty())
}

func TestMicrovmClusterV1alpha2_ValidateUpdate(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()
	w := &webhook.MicrovmClusterV1alpha2{}

	oldCluster := &infrav1alpha2.MicrovmCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: infrav1alpha2.MicrovmClusterSpec{
			Placement: infrav1alpha2.Placement{
				StaticPool: &infrav1alpha2.StaticPoolPlacement{
					Hosts: []infrav1alpha2.MicrovmHost{{Endpoint: "127.0.0.1:9090"}},
				},
			},
		},
	}
	newCluster := oldCluster.DeepCopy()
	warnings, err := w.ValidateUpdate(ctx, oldCluster, newCluster)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(warnings).To(BeEmpty())
}

func TestMicrovmMachineV1alpha2_ValidateCreate(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()
	w := &webhook.MicrovmMachineV1alpha2{}

	machine := &infrav1alpha2.MicrovmMachine{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec:       infrav1alpha2.MicrovmMachineSpec{ProviderID: pointer.String("microvm://host/id")},
	}
	warnings, err := w.ValidateCreate(ctx, machine)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(warnings).To(BeEmpty())
}

func TestMicrovmMachineV1alpha2_ValidateUpdate(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()
	w := &webhook.MicrovmMachineV1alpha2{}

	oldMachine := &infrav1alpha2.MicrovmMachine{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec:       infrav1alpha2.MicrovmMachineSpec{ProviderID: pointer.String("microvm://host/id")},
	}

	t.Run("allows update when spec unchanged", func(t *testing.T) {
		newMachine := oldMachine.DeepCopy()
		warnings, err := w.ValidateUpdate(ctx, oldMachine, newMachine)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(warnings).To(BeEmpty())
	})

	t.Run("rejects update when spec changed (immutable)", func(t *testing.T) {
		newMachine := oldMachine.DeepCopy()
		newMachine.Spec.ProviderID = pointer.String("microvm://other/id")
		warnings, err := w.ValidateUpdate(ctx, oldMachine, newMachine)
		g.Expect(err).To(HaveOccurred())
		g.Expect(warnings).To(BeEmpty())
	})
}

func TestMicrovmMachineV1alpha2_ValidateDelete(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()
	w := &webhook.MicrovmMachineV1alpha2{}

	machine := &infrav1alpha2.MicrovmMachine{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
	}
	warnings, err := w.ValidateDelete(ctx, machine)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(warnings).To(BeEmpty())
}

func TestMicrovmMachineV1alpha2_Default(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()
	w := &webhook.MicrovmMachineV1alpha2{}

	machine := &infrav1alpha2.MicrovmMachine{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: infrav1alpha2.MicrovmMachineSpec{
			VMSpec: microvm.VMSpec{
				NetworkInterfaces: []microvm.NetworkInterface{
					{GuestDeviceName: "eth0", Type: microvm.IfaceTypeMacvtap},
				},
			},
		},
	}
	err := w.Default(ctx, machine)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(machine.Spec.NetworkInterfaces).To(HaveLen(1))
	g.Expect(machine.Spec.NetworkInterfaces[0].GuestMAC).NotTo(BeEmpty())
}

func TestMicrovmMachineTemplateV1alpha2_ValidateCreate(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()
	w := &webhook.MicrovmMachineTemplateV1alpha2{}

	tmpl := &infrav1alpha2.MicrovmMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: infrav1alpha2.MicrovmMachineTemplateSpec{
			Template: infrav1alpha2.MicrovmMachineTemplateResource{
				Spec: infrav1alpha2.MicrovmMachineSpec{},
			},
		},
	}
	warnings, err := w.ValidateCreate(ctx, tmpl)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(warnings).To(BeEmpty())
}

func TestMicrovmMachineTemplateV1alpha2_ValidateDelete(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()
	w := &webhook.MicrovmMachineTemplateV1alpha2{}

	tmpl := &infrav1alpha2.MicrovmMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
	}
	warnings, err := w.ValidateDelete(ctx, tmpl)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(warnings).To(BeEmpty())
}

func TestMicrovmMachineTemplateV1alpha2_ValidateUpdate(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()
	w := &webhook.MicrovmMachineTemplateV1alpha2{}

	oldTmpl := &infrav1alpha2.MicrovmMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: infrav1alpha2.MicrovmMachineTemplateSpec{
			Template: infrav1alpha2.MicrovmMachineTemplateResource{Spec: infrav1alpha2.MicrovmMachineSpec{}},
		},
	}
	newTmpl := oldTmpl.DeepCopy()
	warnings, err := w.ValidateUpdate(ctx, oldTmpl, newTmpl)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(warnings).To(BeEmpty())
}

// Ensure v1alpha2 types are registered and can be used with the scheme (e.g. for webhook tests).
func TestV1alpha2SchemeRegistration(t *testing.T) {
	g := NewWithT(t)
	scheme := runtime.NewScheme()
	err := infrav1alpha2.AddToScheme(scheme)
	g.Expect(err).NotTo(HaveOccurred())
	// Spot-check that types are known
	g.Expect(scheme.Recognizes(infrav1alpha2.GroupVersion.WithKind("MicrovmCluster"))).To(BeTrue())
	g.Expect(scheme.Recognizes(infrav1alpha2.GroupVersion.WithKind("MicrovmMachine"))).To(BeTrue())
	g.Expect(scheme.Recognizes(infrav1alpha2.GroupVersion.WithKind("MicrovmMachineTemplate"))).To(BeTrue())
}
