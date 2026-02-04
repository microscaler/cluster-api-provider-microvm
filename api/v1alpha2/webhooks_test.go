// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestAggregateObjErrors(t *testing.T) {
	g := NewWithT(t)

	gk := schema.GroupKind{Group: "infrastructure.cluster.x-k8s.io", Kind: "MicrovmCluster"}
	name := "test-cluster"

	t.Run("returns nil when no errors", func(t *testing.T) {
		err := aggregateObjErrors(gk, name, nil)
		g.Expect(err).To(BeNil())
		err = aggregateObjErrors(gk, name, field.ErrorList{})
		g.Expect(err).To(BeNil())
	})

	t.Run("returns Invalid error when errors present", func(t *testing.T) {
		errs := field.ErrorList{
			field.Forbidden(field.NewPath("spec", "placement"), "placement required"),
		}
		err := aggregateObjErrors(gk, name, errs)
		g.Expect(err).NotTo(BeNil())
		g.Expect(err.Error()).To(ContainSubstring("placement required"))
		g.Expect(err.Error()).To(ContainSubstring(name))
	})
}
