package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/installer/pkg/types"
	"github.com/openshift/installer/pkg/types/featuregates"
)

// ValidateGCPLabelsTagsFeature validates UserTags and UserLabels can be configured
// only when GCPLabelsTags featureGate is enabled.
func ValidateGCPLabelsTagsFeature(featureGates featuregates.FeatureGate, config *types.InstallConfig) field.ErrorList {
	allErrs := field.ErrorList{}
	if config == nil || config.GCP == nil {
		return allErrs
	}

	errMsg := "featureGate GCPLabelsTags must be enabled with CustomNoUpgrade featureSet or the TechPreviewNoUpgrade featureSet must be enabled to use this field"
	if len(config.GCP.UserTags) > 0 && !featureGates.Enabled(configv1.FeatureGateGCPLabelsTags) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("platform", "gcp", "userTags"), errMsg))
	}
	if len(config.GCP.UserLabels) > 0 && !featureGates.Enabled(configv1.FeatureGateGCPLabelsTags) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("platform", "gcp", "userLabels"), errMsg))
	}

	return allErrs
}
