// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package v1alpha1

import (
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1beta2 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

func TestMicrovmCluster_Hub_and_Conditions(t *testing.T) {
	g := NewWithT(t)

	cluster := &MicrovmCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "default"},
	}
	// Hub() is a no-op; just ensure it doesn't panic
	cluster.Hub()

	g.Expect(cluster.GetConditions()).To(BeNil())
	conds := clusterv1beta2.Conditions{
		{Type: "Ready", Status: corev1.ConditionTrue, LastTransitionTime: metav1.Now()},
	}
	cluster.SetConditions(conds)
	g.Expect(cluster.GetConditions()).To(Equal(conds))
	g.Expect(cluster.GetV1Beta1Conditions()).To(Equal(conds))
	cluster.SetV1Beta1Conditions(nil)
	g.Expect(cluster.GetConditions()).To(BeNil())
}

func TestMicrovmMachine_Hub_and_Conditions(t *testing.T) {
	g := NewWithT(t)

	machine := &MicrovmMachine{
		ObjectMeta: metav1.ObjectMeta{Name: "m1", Namespace: "default"},
	}
	machine.Hub()

	g.Expect(machine.GetConditions()).To(BeNil())
	conds := clusterv1beta2.Conditions{
		{Type: "Ready", Status: corev1.ConditionTrue, LastTransitionTime: metav1.Now()},
	}
	machine.SetConditions(conds)
	g.Expect(machine.GetConditions()).To(Equal(conds))
	machine.SetV1Beta1Conditions(conds)
	g.Expect(machine.GetV1Beta1Conditions()).To(Equal(conds))
}

func TestMicrovmMachineTemplate_Hub(t *testing.T) {
	g := NewWithT(t)

	tmpl := &MicrovmMachineTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "t1", Namespace: "default"},
	}
	tmpl.Hub()
	g.Expect(tmpl.Name).To(Equal("t1"))
}
