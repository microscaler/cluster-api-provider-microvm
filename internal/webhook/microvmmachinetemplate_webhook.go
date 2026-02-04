// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package webhook

import (
	// "fmt"
	"context"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	infrav1 "github.com/liquidmetal-dev/cluster-api-provider-microvm/api/v1alpha1"
	infrav1alpha2 "github.com/liquidmetal-dev/cluster-api-provider-microvm/api/v1alpha2"
)

var _ = logf.Log.WithName("microvmmachinetemplate-resource")

type MicrovmMachineTemplate struct{}


func (r *MicrovmMachineTemplate) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&infrav1.MicrovmMachineTemplate{}).
		WithValidator(r).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1alpha1-microvmmachinetemplate,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=microvmmachinetemplates,versions=v1alpha1,name=validation.microvmmachinetemplate.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1

var _ webhook.CustomValidator = &MicrovmMachineTemplate{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *MicrovmMachineTemplate) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *MicrovmMachineTemplate) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *MicrovmMachineTemplate) ValidateUpdate(_ context.Context, _ runtime.Object, newObj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// MicrovmMachineTemplateV1alpha2 handles v1alpha2 MicrovmMachineTemplate webhooks.
type MicrovmMachineTemplateV1alpha2 struct{}

func (r *MicrovmMachineTemplateV1alpha2) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&infrav1alpha2.MicrovmMachineTemplate{}).
		WithValidator(r).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1alpha2-microvmmachinetemplate,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=microvmmachinetemplates,versions=v1alpha2,name=validation.microvmmachinetemplate.v1alpha2.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1

var _ webhook.CustomValidator = &MicrovmMachineTemplateV1alpha2{}

func (r *MicrovmMachineTemplateV1alpha2) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (r *MicrovmMachineTemplateV1alpha2) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (r *MicrovmMachineTemplateV1alpha2) ValidateUpdate(_ context.Context, _ runtime.Object, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}
