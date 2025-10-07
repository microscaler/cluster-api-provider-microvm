// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package webhook

import (
	"fmt"
	"context"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	// "k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	infrav1 "github.com/liquidmetal-dev/cluster-api-provider-microvm/api/v1alpha1"
)

var _ = logf.Log.WithName("mvmcluster-resource")


type MicrovmCluster struct{}

func (r *MicrovmCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&infrav1.MicrovmCluster{}).
		WithValidator(r).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1alpha1-microvmcluster,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=microvmclusters,versions=v1alpha1,name=validation.microvmcluster.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1
// +kubebuilder:webhook:verbs=create;update,path=/mutate-infrastructure-cluster-x-k8s-io-v1alpha1-microvmcluster,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=microvmclusters,versions=v1alpha1,name=default.microvmcluster.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1

var (
	_ webhook.CustomValidator = &MicrovmCluster{}
	_ webhook.CustomDefaulter = &MicrovmCluster{}
)

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *MicrovmCluster) ValidateCreate(_ context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	cluster, ok := obj.(*infrav1.MicrovmCluster)
	if !ok {
		return warnings, apierrors.NewBadRequest(fmt.Sprintf("expected a MicrovmCluster but got %T", obj))
	}

	allErrs := cluster.Spec.Placement.Validate()
	if len(allErrs) > 0 {
		warnings = append(warnings, fmt.Sprintf("cannot create microvm cluster %s", cluster.GetName()))
		return warnings, apierrors.NewInvalid(
			cluster.GroupVersionKind().GroupKind(),
			cluster.Name,
			allErrs,
		)
	}

	return warnings, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *MicrovmCluster) ValidateDelete(_ context.Context, _ runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *MicrovmCluster) ValidateUpdate(_ context.Context, _ runtime.Object, newObj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// Default satisfies the defaulting webhook interface.
func (r *MicrovmCluster) Default(_ context.Context, obj runtime.Object) error {
	_, ok := obj.(*infrav1.MicrovmCluster)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a MicrovmCluster but got a %T", obj))
	}
	
	return nil
}