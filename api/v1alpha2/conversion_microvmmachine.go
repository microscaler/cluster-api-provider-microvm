// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1 "github.com/liquidmetal-dev/cluster-api-provider-microvm/api/v1alpha1"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	clusterv1beta2 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

// ConvertTo converts this MicrovmMachine to the Hub version (v1alpha1).
func (m *MicrovmMachine) ConvertTo(dst conversion.Hub) error {
	hub := dst.(*infrav1.MicrovmMachine)
	hub.ObjectMeta = m.ObjectMeta
	hub.Spec.VMSpec = m.Spec.VMSpec
	hub.Spec.SSHPublicKeys = m.Spec.SSHPublicKeys
	hub.Spec.ProviderID = m.Spec.ProviderID
	hub.Status.Ready = m.Status.Ready
	hub.Status.VMState = m.Status.VMState
	hub.Status.Addresses = machineAddressesToV1Beta1(m.Status.Addresses)
	hub.Status.FailureReason = m.Status.FailureReason
	hub.Status.FailureMessage = m.Status.FailureMessage
	hub.Status.Conditions = m.Status.Conditions
	return nil
}

// ConvertFrom converts from the Hub version (v1alpha1) to this version.
func (m *MicrovmMachine) ConvertFrom(src conversion.Hub) error {
	hub := src.(*infrav1.MicrovmMachine)
	m.ObjectMeta = hub.ObjectMeta
	m.Spec.VMSpec = hub.Spec.VMSpec
	m.Spec.SSHPublicKeys = hub.Spec.SSHPublicKeys
	m.Spec.ProviderID = hub.Spec.ProviderID
	m.Status.Ready = hub.Status.Ready
	m.Status.VMState = hub.Status.VMState
	m.Status.Addresses = machineAddressesFromV1Beta1(hub.Status.Addresses)
	m.Status.FailureReason = hub.Status.FailureReason
	m.Status.FailureMessage = hub.Status.FailureMessage
	m.Status.Conditions = hub.Status.Conditions
	return nil
}

func machineAddressesToV1Beta1(s []clusterv1beta2.MachineAddress) []clusterv1beta1.MachineAddress {
	if s == nil {
		return nil
	}
	out := make([]clusterv1beta1.MachineAddress, len(s))
	for i := range s {
		out[i] = clusterv1beta1.MachineAddress{
			Type:    clusterv1beta1.MachineAddressType(s[i].Type),
			Address: s[i].Address,
		}
	}
	return out
}

func machineAddressesFromV1Beta1(s []clusterv1beta1.MachineAddress) []clusterv1beta2.MachineAddress {
	if s == nil {
		return nil
	}
	out := make([]clusterv1beta2.MachineAddress, len(s))
	for i := range s {
		out[i] = clusterv1beta2.MachineAddress{
			Type:    clusterv1beta2.MachineAddressType(s[i].Type),
			Address: s[i].Address,
		}
	}
	return out
}
