// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package v1alpha2

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	infrav1 "github.com/liquidmetal-dev/cluster-api-provider-microvm/api/v1alpha1"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/core/v1beta1"
	clusterv1beta2 "sigs.k8s.io/cluster-api/api/core/v1beta2"
)

// ConvertTo converts this MicrovmMachineTemplate to the Hub version (v1alpha1).
func (m *MicrovmMachineTemplate) ConvertTo(dst conversion.Hub) error {
	hub := dst.(*infrav1.MicrovmMachineTemplate)
	hub.ObjectMeta = m.ObjectMeta
	hub.Spec.Template.ObjectMeta = convertObjectMetaToV1Beta1(m.Spec.Template.ObjectMeta)
	hub.Spec.Template.Spec.VMSpec = m.Spec.Template.Spec.VMSpec
	hub.Spec.Template.Spec.SSHPublicKeys = m.Spec.Template.Spec.SSHPublicKeys
	hub.Spec.Template.Spec.ProviderID = m.Spec.Template.Spec.ProviderID
	return nil
}

// ConvertFrom converts from the Hub version (v1alpha1) to this version.
func (m *MicrovmMachineTemplate) ConvertFrom(src conversion.Hub) error {
	hub := src.(*infrav1.MicrovmMachineTemplate)
	m.ObjectMeta = hub.ObjectMeta
	m.Spec.Template.ObjectMeta = convertObjectMetaFromV1Beta1(hub.Spec.Template.ObjectMeta)
	m.Spec.Template.Spec.VMSpec = hub.Spec.Template.Spec.VMSpec
	m.Spec.Template.Spec.SSHPublicKeys = hub.Spec.Template.Spec.SSHPublicKeys
	m.Spec.Template.Spec.ProviderID = hub.Spec.Template.Spec.ProviderID
	return nil
}

func convertObjectMetaToV1Beta1(meta clusterv1beta2.ObjectMeta) clusterv1beta1.ObjectMeta {
	return clusterv1beta1.ObjectMeta{
		Labels:      meta.Labels,
		Annotations: meta.Annotations,
	}
}

func convertObjectMetaFromV1Beta1(meta clusterv1beta1.ObjectMeta) clusterv1beta2.ObjectMeta {
	return clusterv1beta2.ObjectMeta{
		Labels:      meta.Labels,
		Annotations: meta.Annotations,
	}
}
