package featuregates

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"

	configv1 "github.com/openshift/api/config/v1"
)

// FeatureGate indicates whether a given feature is enabled or not
// This interface is directly taken from openshift/library-go and modified as required.
type FeatureGate interface {
	// Enabled returns true if the key is enabled.
	Enabled(key configv1.FeatureGateName) bool
}

// featureGate holds list of enabled and disabled features of a featureSet.
type featureGate struct {
	enabled  sets.Set[configv1.FeatureGateName]
	disabled sets.Set[configv1.FeatureGateName]
}

// NewFeatureGate initializes the enabled and disabled features list.
func NewFeatureGate(enabled, disabled []configv1.FeatureGateName) FeatureGate {
	return &featureGate{
		enabled:  sets.New[configv1.FeatureGateName](enabled...),
		disabled: sets.New[configv1.FeatureGateName](disabled...),
	}
}

// Enabled returns true when a featureGate is enabled.
func (f *featureGate) Enabled(key configv1.FeatureGateName) bool { return f.enabled.Has(key) }

// GenerateCustomFeatures generates the custom feature gates from the install-config.
func GenerateCustomFeatures(features []string) (*configv1.CustomFeatureGates, error) {
	if len(features) == 0 {
		return new(configv1.CustomFeatureGates), nil
	}

	customFeatures := new(configv1.CustomFeatureGates)
	knownFeatures := getKnownFeatures()

	var errs []error
	for _, feature := range features {
		featureName, enabled, err := parseCustomFeatureGate(knownFeatures, feature)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if enabled {
			customFeatures.Enabled = append(customFeatures.Enabled, featureName)
		} else {
			customFeatures.Disabled = append(customFeatures.Disabled, featureName)
		}
	}
	if len(errs) != 0 {
		return nil, fmt.Errorf("failed to parse custom features: %s", errs)
	}

	return customFeatures, nil
}

// GetFeatureGates returns the list of enabled and disabled features for a given featureSet.
func GetFeatureGates(featureSet configv1.FeatureSet, featureGates ...string) (FeatureGate, error) {
	switch featureSet {
	case "":
		return &featureGate{
			enabled:  sets.New[configv1.FeatureGateName](),
			disabled: sets.New[configv1.FeatureGateName](),
		}, nil
	case configv1.TechPreviewNoUpgrade:
		fg := &featureGate{
			enabled:  sets.New[configv1.FeatureGateName](),
			disabled: sets.New[configv1.FeatureGateName](),
		}
		features := configv1.FeatureSets[configv1.TechPreviewNoUpgrade]
		knownFeatures := getKnownFeatures()
		for _, f := range features.Enabled {
			fg.enabled.Insert(f.FeatureGateAttributes.Name)
		}
		fg.disabled = knownFeatures.Difference(fg.enabled)
		return fg, nil
	case configv1.CustomNoUpgrade:
		customFeatures, err := GenerateCustomFeatures(featureGates)
		if err != nil {
			return nil, err
		}
		knownFeatures := getKnownFeatures()
		enabledFeatures := sets.New[configv1.FeatureGateName](customFeatures.Enabled...)
		return &featureGate{
			enabled:  enabledFeatures,
			disabled: knownFeatures.Difference(enabledFeatures),
		}, nil
	case configv1.LatencySensitive:
		fg := &featureGate{
			enabled:  sets.New[configv1.FeatureGateName](),
			disabled: sets.New[configv1.FeatureGateName](),
		}
		features := configv1.FeatureSets[configv1.LatencySensitive]
		knownFeatures := getKnownFeatures()
		for _, f := range features.Enabled {
			fg.enabled.Insert(f.FeatureGateAttributes.Name)
		}
		fg.disabled = knownFeatures.Difference(fg.enabled)
		return fg, nil
	}
	return nil, fmt.Errorf("unknown featureSet %s", featureSet)
}

// getKnownFeatures returns the list of all known FeatureGates.
func getKnownFeatures() sets.Set[configv1.FeatureGateName] {
	knownFeatures := sets.New[configv1.FeatureGateName]()

	for _, featureSet := range configv1.FeatureSets {
		for _, feature := range featureSet.Enabled {
			knownFeatures.Insert(feature.FeatureGateAttributes.Name)
		}
		for _, feature := range featureSet.Disabled {
			knownFeatures.Insert(feature.FeatureGateAttributes.Name)
		}
	}

	return knownFeatures
}

// parseCustomFeatureGates parses the custom feature gate string into the feature name and whether it is enabled.
// The expected format is <FeatureName>=<Enabled>.
func parseCustomFeatureGate(knownFeatures sets.Set[configv1.FeatureGateName], rawFeature string) (configv1.FeatureGateName, bool, error) {
	var featureName string
	var enabled bool

	featureParts := strings.Split(rawFeature, "=")
	if len(featureParts) != 2 {
		return "", false, fmt.Errorf("feature not in expected format %s", rawFeature)
	}
	featureName = featureParts[0]

	var err error
	if !knownFeatures.Has(configv1.FeatureGateName(featureName)) {
		return "", false, fmt.Errorf("unsupported \"%s\" featureGate configured", rawFeature)
	}
	enabled, err = strconv.ParseBool(featureParts[1])
	if err != nil {
		return "", false, fmt.Errorf("feature not in expected format %s, could not parse boolean value: %w", rawFeature, err)
	}

	return configv1.FeatureGateName(featureName), enabled, nil
}
