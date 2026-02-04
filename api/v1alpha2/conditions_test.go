// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

func TestMicrovmCluster_GetConditions_SetConditions(t *testing.T) {
	g := NewWithT(t)

	cluster := &MicrovmCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "c1", Namespace: "default"},
	}

	g.Expect(cluster.GetConditions()).To(BeNil())
	conds := clusterv1.Conditions{
		{Type: "Ready", Status: corev1.ConditionTrue, LastTransitionTime: metav1.Now()},
	}
	cluster.SetConditions(conds)
	g.Expect(cluster.GetConditions()).To(Equal(conds))
}

func TestMicrovmMachine_GetConditions_SetConditions(t *testing.T) {
	g := NewWithT(t)

	machine := &MicrovmMachine{
		ObjectMeta: metav1.ObjectMeta{Name: "m1", Namespace: "default"},
	}

	g.Expect(machine.GetConditions()).To(BeNil())
	conds := clusterv1.Conditions{
		{Type: "Ready", Status: corev1.ConditionTrue, LastTransitionTime: metav1.Now()},
	}
	machine.SetConditions(conds)
	g.Expect(machine.GetConditions()).To(Equal(conds))
}
