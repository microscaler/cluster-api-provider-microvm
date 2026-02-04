// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/liquidmetal-dev/controller-pkg/types/microvm"
)

func TestSetDefaults_NetworkInterface(t *testing.T) {
	g := NewWithT(t)

	t.Run("sets GuestMAC when empty", func(t *testing.T) {
		obj := &microvm.NetworkInterface{GuestMAC: ""}
		SetDefaults_NetworkInterface(obj)
		g.Expect(obj.GuestMAC).NotTo(BeEmpty())
		g.Expect(obj.GuestMAC).To(MatchRegexp(`^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$`))
	})

	t.Run("leaves GuestMAC unchanged when already set", func(t *testing.T) {
		existing := "aa:bb:cc:dd:ee:ff"
		obj := &microvm.NetworkInterface{GuestMAC: existing}
		SetDefaults_NetworkInterface(obj)
		g.Expect(obj.GuestMAC).To(Equal(existing))
	})
}
