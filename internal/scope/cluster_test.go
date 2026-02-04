// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package scope_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"k8s.io/klog/v2/klogr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	infrav1 "github.com/liquidmetal-dev/cluster-api-provider-microvm/api/v1alpha1"
	"github.com/liquidmetal-dev/cluster-api-provider-microvm/internal/scope"
)

func TestNewClusterScope_Validation(t *testing.T) {
	g := NewWithT(t)
	scheme, err := setupScheme()
	g.Expect(err).NotTo(HaveOccurred())

	cluster := newCluster("c1", []string{"fd1"})
	mvmCluster := newMicrovmClusterWithSpec("c1", infrav1.MicrovmClusterSpec{
		Placement: infrav1.Placement{
			StaticPool: &infrav1.StaticPoolPlacement{
				Hosts: []infrav1.MicrovmHost{{Endpoint: "127.0.0.1:9090"}},
			},
		},
	})
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster, mvmCluster).Build()

	_, err = scope.NewClusterScope(nil, mvmCluster, client)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("cluster"))

	_, err = scope.NewClusterScope(cluster, nil, client)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("microvm cluster"))

	_, err = scope.NewClusterScope(cluster, mvmCluster, nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("client"))
}

func TestNewClusterScope_Success(t *testing.T) {
	g := NewWithT(t)
	scheme, err := setupScheme()
	g.Expect(err).NotTo(HaveOccurred())

	cluster := newCluster("c1", []string{"fd1"})
	mvmCluster := newMicrovmClusterWithSpec("c1", infrav1.MicrovmClusterSpec{
		Placement: infrav1.Placement{
			StaticPool: &infrav1.StaticPoolPlacement{
				Hosts: []infrav1.MicrovmHost{{Endpoint: "127.0.0.1:9090"}},
			},
		},
	})
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster, mvmCluster).Build()

	cs, err := scope.NewClusterScope(cluster, mvmCluster, client)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cs).NotTo(BeNil())

	g.Expect(cs.Name()).To(Equal(mvmCluster.Name))
	g.Expect(cs.Namespace()).To(Equal(mvmCluster.Namespace))
	g.Expect(cs.ClusterName()).To(Equal(cluster.Name))
	g.Expect(cs.ControllerName()).To(Equal("microvm-manager"))
	g.Expect(cs.Placement()).To(Equal(mvmCluster.Spec.Placement))
}

func TestClusterScope_WithOptions(t *testing.T) {
	g := NewWithT(t)
	scheme, err := setupScheme()
	g.Expect(err).NotTo(HaveOccurred())

	cluster := newCluster("c1", nil)
	mvmCluster := newMicrovmClusterWithSpec("c1", infrav1.MicrovmClusterSpec{
		Placement: infrav1.Placement{
			StaticPool: &infrav1.StaticPoolPlacement{
				Hosts: []infrav1.MicrovmHost{{Endpoint: "127.0.0.1:9090"}},
			},
		},
	})
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster, mvmCluster).Build()

	cs, err := scope.NewClusterScope(cluster, mvmCluster, client,
		scope.WithClusterLogger(klogr.New()),
		scope.WithClusterControllerName("custom-controller"),
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(cs.ControllerName()).To(Equal("custom-controller"))
}
