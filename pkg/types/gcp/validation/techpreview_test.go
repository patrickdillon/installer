package validation

import (
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/installer/pkg/types"
	"github.com/openshift/installer/pkg/types/featuregates"
	"github.com/openshift/installer/pkg/types/gcp"
)

func TestValidateGCPLabelsTagsFeature(t *testing.T) {
	cases := []struct {
		name                  string
		config                *types.InstallConfig
		enabledFeatures       []configv1.FeatureGateName
		disabledFeatures      []configv1.FeatureGateName
		disallowedFieldsCount int
	}{
		{
			name: "CustomNoUpdate featureSet, GCPLabelsTags featureGate enabled",
			config: &types.InstallConfig{
				FeatureSet: configv1.CustomNoUpgrade,
				FeatureGates: []string{
					"GCPLabelsTags=True",
					"OtherFeature=False",
				},
				Platform: types.Platform{
					GCP: &gcp.Platform{
						UserTags: []gcp.UserTag{
							{ParentID: "123", Key: "key", Value: "value"},
						},
						UserLabels: []gcp.UserLabel{
							{Key: "key", Value: "value"},
						},
					},
				},
			},
			enabledFeatures:       []configv1.FeatureGateName{"GCPLabelsTags"},
			disabledFeatures:      []configv1.FeatureGateName{"OtherFeature"},
			disallowedFieldsCount: 0,
		},
		{
			name: "CustomNoUpdate featureSet, GCPLabelsTags featureGate disabled",
			config: &types.InstallConfig{
				FeatureSet: configv1.CustomNoUpgrade,
				FeatureGates: []string{
					"GCPLabelsTags=False",
					"OtherFeature=False",
				},
				Platform: types.Platform{
					GCP: &gcp.Platform{
						UserTags: []gcp.UserTag{
							{ParentID: "123", Key: "key", Value: "value"},
						},
						UserLabels: []gcp.UserLabel{
							{Key: "key", Value: "value"},
						},
					},
				},
			},
			enabledFeatures:       []configv1.FeatureGateName{},
			disabledFeatures:      []configv1.FeatureGateName{"GCPLabelsTags", "OtherFeature"},
			disallowedFieldsCount: 2,
		},
		{
			name: "TechPreviewNoUpgrade featureSet",
			config: &types.InstallConfig{
				FeatureSet: configv1.TechPreviewNoUpgrade,
				Platform: types.Platform{
					GCP: &gcp.Platform{
						UserTags: []gcp.UserTag{
							{ParentID: "123", Key: "key", Value: "value"},
						},
						UserLabels: []gcp.UserLabel{
							{Key: "key", Value: "value"},
						},
					},
				},
			},
			enabledFeatures:       []configv1.FeatureGateName{"GCPLabelsTags"},
			disabledFeatures:      []configv1.FeatureGateName{},
			disallowedFieldsCount: 0,
		},
		{
			name: "No featureSet enabled, labels and tags defined",
			config: &types.InstallConfig{
				Platform: types.Platform{
					GCP: &gcp.Platform{
						UserTags: []gcp.UserTag{
							{ParentID: "123", Key: "key", Value: "value"},
						},
						UserLabels: []gcp.UserLabel{
							{Key: "key", Value: "value"},
						},
					},
				},
			},
			enabledFeatures:       []configv1.FeatureGateName{},
			disabledFeatures:      []configv1.FeatureGateName{},
			disallowedFieldsCount: 2,
		},
		{
			name:                  "No featureSet enabled",
			config:                &types.InstallConfig{},
			enabledFeatures:       []configv1.FeatureGateName{},
			disabledFeatures:      []configv1.FeatureGateName{},
			disallowedFieldsCount: 0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			featureGates := featuregates.NewFeatureGate(tc.enabledFeatures, tc.disabledFeatures)
			fields := ValidateGCPLabelsTagsFeature(featureGates, tc.config)

			if len(fields) != tc.disallowedFieldsCount {
				t.Errorf("Got: %v, disallowedFields: %v", fields, tc.disallowedFieldsCount)
			}
		})
	}
}
