package featuregates

import (
	"testing"

	configv1 "github.com/openshift/api/config/v1"
)

func TestGenerateCustomFeatures(t *testing.T) {
	cases := []struct {
		name             string
		featureGates     []string
		enabledFeatures  []string
		disabledFeatures []string
		wantErr          bool
	}{
		{
			name:             "CustomNoUpdate featureSet",
			featureGates:     []string{"OpenShiftPodSecurityAdmission=True", "CloudDualStackNodeIPs=False"},
			enabledFeatures:  []string{"OpenShiftPodSecurityAdmission"},
			disabledFeatures: []string{"CloudDualStackNodeIPs"},
			wantErr:          false,
		},
		{
			name:             "CustomNoUpdate featureSet, no features configured",
			featureGates:     nil,
			enabledFeatures:  []string{},
			disabledFeatures: []string{},
			wantErr:          false,
		},
		{
			name:             "Invalid featureGates configuration",
			featureGates:     []string{"OpenShiftPodSecurityAdmission", "CloudDualStackNodeIPs=False"},
			enabledFeatures:  []string{"OpenShiftPodSecurityAdmission"},
			disabledFeatures: []string{"CloudDualStackNodeIPs"},
			wantErr:          true,
		},
		{
			name:             "Unsupported featureGate",
			featureGates:     []string{"OpenShiftPodSecurityAdmission=True", "CustomFeature=False"},
			enabledFeatures:  []string{"OpenShiftPodSecurityAdmission"},
			disabledFeatures: []string{"CustomFeature"},
			wantErr:          true,
		},
		{
			name:             "Invalid featureGate value",
			featureGates:     []string{"OpenShiftPodSecurityAdmission=yes", "CustomFeature=no"},
			enabledFeatures:  []string{"OpenShiftPodSecurityAdmission"},
			disabledFeatures: []string{"CustomFeature"},
			wantErr:          true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			featureGates, err := GenerateCustomFeatures(tc.featureGates)

			if (err != nil) != tc.wantErr {
				t.Errorf("Got: %v, wantErr: %v", err, tc.wantErr)
			}

			if err == nil {
				features := NewFeatureGate(featureGates.Enabled, featureGates.Disabled)

				for _, f := range tc.enabledFeatures {
					if !features.Enabled(configv1.FeatureGateName(f)) {
						t.Errorf("Expcted %s feature to be enabled", f)
					}
				}

				for _, f := range tc.disabledFeatures {
					if features.Enabled(configv1.FeatureGateName(f)) {
						t.Errorf("Expected %s feature to be disabled", f)
					}
				}
			}
		})
	}
}

func TestGetFeatureGates(t *testing.T) {
	cases := []struct {
		name            string
		featureSet      configv1.FeatureSet
		featureGates    []string
		enabledFeatures []string
		wantErr         bool
	}{
		{
			name:            "TechPreviewNoUpgrade featureSet",
			featureSet:      configv1.TechPreviewNoUpgrade,
			featureGates:    nil,
			enabledFeatures: []string{"ExternalCloudProvider"},
			wantErr:         false,
		},
		{
			name:            "CustomNoUpdate featureSet",
			featureSet:      configv1.CustomNoUpgrade,
			featureGates:    []string{"OpenShiftPodSecurityAdmission=True", "CloudDualStackNodeIPs=False"},
			enabledFeatures: []string{"OpenShiftPodSecurityAdmission"},
			wantErr:         false,
		},
		{
			name:            "CustomNoUpdate featureSet, invalid featureGate",
			featureSet:      configv1.CustomNoUpgrade,
			featureGates:    []string{"OpenShiftPodSecurityAdmission=True", "CustomFeature=False"},
			enabledFeatures: nil,
			wantErr:         true,
		},
		{
			name:            "LatencySensitive featureSet",
			featureSet:      configv1.LatencySensitive,
			featureGates:    nil,
			enabledFeatures: []string{"OpenShiftPodSecurityAdmission"},
			wantErr:         false,
		},
		{
			name:            "Unknown featureSet",
			featureSet:      configv1.FeatureSet("Unknown"),
			featureGates:    []string{"CustomFeature1=True", "CustomFeature2=False"},
			enabledFeatures: nil,
			wantErr:         true,
		},
		{
			name:            "featureSet not configured",
			featureSet:      "",
			featureGates:    nil,
			enabledFeatures: nil,
			wantErr:         false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			featureGates, err := GetFeatureGates(tc.featureSet, tc.featureGates...)

			if (err != nil) != tc.wantErr {
				t.Errorf("Got: %v, wantErr: %v", err, tc.wantErr)
			}

			for _, f := range tc.enabledFeatures {
				if !featureGates.Enabled(configv1.FeatureGateName(f)) {
					t.Errorf("Expected %s feature to be in enabled list", f)
				}
			}
		})
	}
}
