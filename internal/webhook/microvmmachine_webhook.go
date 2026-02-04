// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package webhook

import (
	"fmt"
	"reflect"
	"context"
	"k8s.io/apimachinery/pkg/runtime"
	// "k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	infrav1 "github.com/liquidmetal-dev/cluster-api-provider-microvm/api/v1alpha1"
	infrav1alpha2 "github.com/liquidmetal-dev/cluster-api-provider-microvm/api/v1alpha2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

var machineLog = logf.Log.WithName("microvmmachine-resource")

type MicrovmMachine struct{}


func (r *MicrovmMachine) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&infrav1.MicrovmMachine{}).
		WithValidator(r).
		WithDefaulter(r, admission.DefaulterRemoveUnknownOrOmitableFields).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1alpha1-microvmmachine,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=microvmmachine,versions=v1alpha1,name=validation.microvmmachine.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1
// +kubebuilder:webhook:verbs=create;update,path=/mutate-infrastructure-cluster-x-k8s-io-v1alpha1-microvmmachine,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=microvmmachine,versions=v1alpha1,name=default.microvmmachine.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1

var (
	_ webhook.CustomValidator = &MicrovmMachine{}
	_ webhook.CustomDefaulter = &MicrovmMachine{}
)

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *MicrovmMachine) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *MicrovmMachine) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *MicrovmMachine) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	newMachine, ok := newObj.(*infrav1.MicrovmMachine)
	if !ok {
		return warnings, apierrors.NewBadRequest(fmt.Sprintf("expected a MicrovmMachine but got %T", newObj))
	}
	oldMachine, ok := oldObj.(*infrav1.MicrovmMachine)
	if !ok {
		return warnings, apierrors.NewBadRequest(fmt.Sprintf("expected a MicrovmMachine but got %T", oldObj))
	}

	machineLog.Info("validate update", "name", newMachine.Name)

	// spec is immutable
	if !reflect.DeepEqual(newMachine.Spec, oldMachine.Spec) {
		return warnings, apierrors.NewBadRequest("microvm machine spec is immutable")
	}

	return warnings, nil
}

// Default satisfies the defaulting webhook interface.
func (r *MicrovmMachine) Default(_ context.Context, obj runtime.Object) error {
	machine, ok := obj.(*infrav1.MicrovmMachine)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a MicrovmMachine but got a %T", obj))
	}

	infrav1.SetObjectDefaults_MicrovmMachine(machine)

	return nil
}

// MicrovmMachineV1alpha2 handles v1alpha2 MicrovmMachine webhooks.
type MicrovmMachineV1alpha2 struct{}

func (r *MicrovmMachineV1alpha2) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&infrav1alpha2.MicrovmMachine{}).
		WithValidator(r).
		WithDefaulter(r, admission.DefaulterRemoveUnknownOrOmitableFields).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1alpha2-microvmmachine,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=microvmmachine,versions=v1alpha2,name=validation.microvmmachine.v1alpha2.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1
// +kubebuilder:webhook:verbs=create;update,path=/mutate-infrastructure-cluster-x-k8s-io-v1alpha2-microvmmachine,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=microvmmachine,versions=v1alpha2,name=default.microvmmachine.v1alpha2.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1

var (
	_ webhook.CustomValidator = &MicrovmMachineV1alpha2{}
	_ webhook.CustomDefaulter = &MicrovmMachineV1alpha2{}
)

func (r *MicrovmMachineV1alpha2) ValidateCreate(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (r *MicrovmMachineV1alpha2) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

func (r *MicrovmMachineV1alpha2) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	newMachine, ok := newObj.(*infrav1alpha2.MicrovmMachine)
	if !ok {
		return warnings, apierrors.NewBadRequest(fmt.Sprintf("expected a MicrovmMachine but got %T", newObj))
	}
	oldMachine, ok := oldObj.(*infrav1alpha2.MicrovmMachine)
	if !ok {
		return warnings, apierrors.NewBadRequest(fmt.Sprintf("expected a MicrovmMachine but got %T", oldObj))
	}
	if !reflect.DeepEqual(newMachine.Spec, oldMachine.Spec) {
		return warnings, apierrors.NewBadRequest("microvm machine spec is immutable")
	}
	return warnings, nil
}

func (r *MicrovmMachineV1alpha2) Default(_ context.Context, obj runtime.Object) error {
	machine, ok := obj.(*infrav1alpha2.MicrovmMachine)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a MicrovmMachine but got a %T", obj))
	}
	infrav1alpha2.SetObjectDefaults_MicrovmMachine(machine)
	return nil
}
