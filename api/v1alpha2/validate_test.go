// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestPlacement_Validate(t *testing.T) {
	g := NewWithT(t)

	t.Run("returns error when StaticPool is nil", func(t *testing.T) {
		p := &Placement{}
		errs := p.Validate()
		g.Expect(errs).To(HaveLen(1))
		g.Expect(errs[0].Type).To(Equal(field.ErrorTypeForbidden))
		g.Expect(errs[0].Field).To(Equal("spec.placement"))
		g.Expect(errs[0].Detail).To(ContainSubstring("placement option"))
	})

	t.Run("returns no errors when StaticPool is set", func(t *testing.T) {
		p := &Placement{
			StaticPool: &StaticPoolPlacement{
				Hosts: []MicrovmHost{{Endpoint: "127.0.0.1:9090"}},
			},
		}
		errs := p.Validate()
		g.Expect(errs).To(BeEmpty())
	})
}

func TestPlacement_IsSet(t *testing.T) {
	g := NewWithT(t)

	t.Run("returns false when StaticPool is nil", func(t *testing.T) {
		p := &Placement{}
		g.Expect(p.IsSet()).To(BeFalse())
	})

	t.Run("returns true when StaticPool is set", func(t *testing.T) {
		p := &Placement{
			StaticPool: &StaticPoolPlacement{Hosts: []MicrovmHost{}},
		}
		g.Expect(p.IsSet()).To(BeTrue())
	})
}
