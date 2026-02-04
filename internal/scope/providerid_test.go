// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package scope_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/liquidmetal-dev/cluster-api-provider-microvm/internal/scope"
)

func TestProviderID_String(t *testing.T) {
	g := NewWithT(t)

	p, err := scope.NewProviderID("microvm://fd1/abc-123")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(p.String()).To(Equal("microvm://fd1/abc-123"))
}

func TestProviderID_Equals(t *testing.T) {
	g := NewWithT(t)

	p1, err := scope.NewProviderID("microvm://fd1/id1")
	g.Expect(err).NotTo(HaveOccurred())
	p2, err := scope.NewProviderID("microvm://fd1/id1")
	g.Expect(err).NotTo(HaveOccurred())
	p3, err := scope.NewProviderID("microvm://fd2/id2")
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(p1.Equals(p2)).To(BeTrue())
	g.Expect(p1.Equals(p3)).To(BeFalse())
}

func TestProviderID_IndexKey(t *testing.T) {
	g := NewWithT(t)

	p, err := scope.NewProviderID("microvm://fd1/instance-id")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(p.IndexKey()).To(Equal("microvm://fd1/instance-id"))
}

func TestGenerateProviderID(t *testing.T) {
	g := NewWithT(t)

	// GenerateProviderID formats as ProviderPrefix + "/" + strings.Join(ids, "/")
	g.Expect(scope.GenerateProviderID("fd1", "id1")).To(Equal("microvm:///fd1/id1"))
	g.Expect(scope.GenerateProviderID("fd1", "seg", "id1")).To(Equal("microvm:///fd1/seg/id1"))
}

func TestNewProviderID_Invalid(t *testing.T) {
	g := NewWithT(t)

	_, err := scope.NewProviderID("")
	g.Expect(err).To(Equal(scope.ErrEmptyProviderID))

	_, err = scope.NewProviderID("no-slashes")
	g.Expect(err).To(Equal(scope.ErrInvalidProviderID))
}
