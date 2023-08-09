package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/openshift/installer/pkg/types"
	"github.com/openshift/installer/pkg/types/gcp"
)

func TestValidatePlatform(t *testing.T) {
	cases := []struct {
		name            string
		platform        *gcp.Platform
		credentialsMode types.CredentialsMode
		valid           bool
	}{
		{
			name: "minimal",
			platform: &gcp.Platform{
				Region: "us-east1",
			},
			valid: true,
		},
		{
			name: "invalid region",
			platform: &gcp.Platform{
				Region: "",
			},
			valid: false,
		},
		{
			name: "valid machine pool",
			platform: &gcp.Platform{
				Region:                 "us-east1",
				DefaultMachinePlatform: &gcp.MachinePool{},
			},
			valid: true,
		},
		{
			name: "valid subnets & network",
			platform: &gcp.Platform{
				Region:             "us-east1",
				Network:            "valid-vpc",
				ComputeSubnet:      "valid-compute-subnet",
				ControlPlaneSubnet: "valid-cp-subnet",
			},
			valid: true,
		},
		{
			name: "missing subnets",
			platform: &gcp.Platform{
				Region:  "us-east1",
				Network: "valid-vpc",
			},
			valid: false,
		},
		{
			name: "subnets missing network",
			platform: &gcp.Platform{
				Region:        "us-east1",
				ComputeSubnet: "valid-compute-subnet",
			},
			valid: false,
		},
		{
			name: "unsupported GCP disk type",
			platform: &gcp.Platform{
				Region: "us-east1",
				DefaultMachinePlatform: &gcp.MachinePool{
					OSDisk: gcp.OSDisk{
						DiskType: "pd-standard",
					},
				},
			},
			valid: false,
		},

		{
			name: "supported GCP disk type",
			platform: &gcp.Platform{
				Region: "us-east1",
				DefaultMachinePlatform: &gcp.MachinePool{
					OSDisk: gcp.OSDisk{
						DiskType: "pd-ssd",
					},
				},
			},
			valid: true,
		},
		{
			name: "GCP valid network project data",
			platform: &gcp.Platform{
				Region:             "us-east1",
				NetworkProjectID:   "valid-network-project",
				ProjectID:          "valid-project",
				Network:            "valid-vpc",
				ComputeSubnet:      "valid-compute-subnet",
				ControlPlaneSubnet: "valid-cp-subnet",
			},
			credentialsMode: types.PassthroughCredentialsMode,
			valid:           true,
		},
		{
			name: "GCP invalid network project missing network",
			platform: &gcp.Platform{
				Region:             "us-east1",
				NetworkProjectID:   "valid-network-project",
				ProjectID:          "valid-project",
				ComputeSubnet:      "valid-compute-subnet",
				ControlPlaneSubnet: "valid-cp-subnet",
			},
			credentialsMode: types.PassthroughCredentialsMode,
			valid:           false,
		},
		{
			name: "GCP invalid network project missing compute subnet",
			platform: &gcp.Platform{
				Region:             "us-east1",
				NetworkProjectID:   "valid-network-project",
				ProjectID:          "valid-project",
				Network:            "valid-vpc",
				ControlPlaneSubnet: "valid-cp-subnet",
			},
			credentialsMode: types.PassthroughCredentialsMode,
			valid:           false,
		},
		{
			name: "GCP invalid network project missing control plane subnet",
			platform: &gcp.Platform{
				Region:           "us-east1",
				NetworkProjectID: "valid-network-project",
				ProjectID:        "valid-project",
				Network:          "valid-vpc",
				ComputeSubnet:    "valid-compute-subnet",
			},
			credentialsMode: types.PassthroughCredentialsMode,
			valid:           false,
		},
		{
			name: "GCP invalid network project bad credentials mode",
			platform: &gcp.Platform{
				Region:             "us-east1",
				NetworkProjectID:   "valid-network-project",
				ProjectID:          "valid-project",
				Network:            "valid-vpc",
				ComputeSubnet:      "valid-compute-subnet",
				ControlPlaneSubnet: "valid-cp-subnet",
			},
			credentialsMode: types.MintCredentialsMode,
			valid:           false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			credentialsMode := tc.credentialsMode
			if credentialsMode == "" {
				credentialsMode = types.MintCredentialsMode
			}

			// the only item currently used is the credentialsMode
			ic := types.InstallConfig{
				CredentialsMode: credentialsMode,
			}

			err := ValidatePlatform(tc.platform, field.NewPath("test-path"), &ic).ToAggregate()
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestValidateUserLabels(t *testing.T) {
	fieldPath := "spec.platform.gcp.UserLabels"
	cases := []struct {
		name       string
		userLabels []gcp.UserLabel
		wantErr    bool
	}{
		{
			name:       "userLabels not configured",
			userLabels: nil,
			wantErr:    false,
		},
		{
			name: "userLabels configured",
			userLabels: []gcp.UserLabel{
				{Key: "key1", Value: "value1"},
				{Key: "key_2", Value: "value_2"},
				{Key: "key-3", Value: "value-3"},
				{Key: "key4_", Value: "value4_"},
				{Key: "key5-", Value: "value5-"},
			},
			wantErr: false,
		},
		{
			name: "userLabels configured is more than max limit",
			userLabels: []gcp.UserLabel{
				{Key: "key11", Value: "value11"}, {Key: "key18", Value: "value18"},
				{Key: "key19", Value: "value19"}, {Key: "key21", Value: "value21"},
				{Key: "key14", Value: "value14"}, {Key: "key22", Value: "value22"},
				{Key: "key25", Value: "value25"}, {Key: "key27", Value: "value27"},
				{Key: "key31", Value: "value31"}, {Key: "key9", Value: "value9"},
				{Key: "key10", Value: "value10"}, {Key: "key15", Value: "value15"},
				{Key: "key28", Value: "value28"}, {Key: "key29", Value: "value29"},
				{Key: "key32", Value: "value32"}, {Key: "key3", Value: "value3"},
				{Key: "key7", Value: "value7"}, {Key: "key17", Value: "value17"},
				{Key: "key20", Value: "value20"}, {Key: "key4", Value: "value4"},
				{Key: "key23", Value: "value23"}, {Key: "key26", Value: "value26"},
				{Key: "key12", Value: "value12"}, {Key: "key33", Value: "value33"},
				{Key: "key1", Value: "value1"}, {Key: "key2", Value: "value2"},
				{Key: "key5", Value: "value5"}, {Key: "key8", Value: "value8"},
				{Key: "key30", Value: "value30"}, {Key: "key6", Value: "value6"},
				{Key: "key13", Value: "value13"}, {Key: "key16", Value: "value16"},
				{Key: "key24", Value: "value24"},
			},
			wantErr: true,
		},
		{
			name:       "userLabels contains key starting a number",
			userLabels: []gcp.UserLabel{{Key: "1key", Value: "1value"}},
			wantErr:    true,
		},
		{
			name:       "userLabels contains key starting a uppercase letter",
			userLabels: []gcp.UserLabel{{Key: "Key", Value: "1value"}},
			wantErr:    true,
		},
		{
			name:       "userLabels contains empty key",
			userLabels: []gcp.UserLabel{{Key: "", Value: "value"}},
			wantErr:    true,
		},
		{
			name: "userLabels contains key length greater than 63",
			userLabels: []gcp.UserLabel{
				{
					Key:   "thisisaverylongkeywithmorethan63characterswhichisnotallowedforgcpresourcelabelkey",
					Value: "value",
				},
			},
			wantErr: true,
		},
		{
			name:       "userLabels contains key with invalid character",
			userLabels: []gcp.UserLabel{{Key: "key/test", Value: "value"}},
			wantErr:    true,
		},
		{
			name: "userLabels contains value length greater than 63",
			userLabels: []gcp.UserLabel{
				{
					Key:   "key",
					Value: "thisisaverylongvaluewithmorethan63characterswhichisnotallowedforgcpresourcelabelvalue",
				},
			},
			wantErr: true,
		},
		{
			name:       "userLabels contains empty value",
			userLabels: []gcp.UserLabel{{Key: "key", Value: ""}},
			wantErr:    true,
		},
		{
			name:       "userLabels contains value with invalid character",
			userLabels: []gcp.UserLabel{{Key: "key", Value: "value*^%"}},
			wantErr:    true,
		},
		{
			name:       "userLabels contains key with prefix kubernetes-io",
			userLabels: []gcp.UserLabel{{Key: "kubernetes-io_cluster", Value: "value"}},
			wantErr:    true,
		},
		{
			name:       "userLabels contains allowed key prefix for_openshift-io",
			userLabels: []gcp.UserLabel{{Key: "for_openshift-io", Value: "gcp"}},
			wantErr:    false,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUserLabels(tt.userLabels, field.NewPath(fieldPath))
			if (len(err) > 0) != tt.wantErr {
				t.Errorf("unexpected error, err: %v", err)
			}
		})
	}
}

func TestValidateUserTags(t *testing.T) {
	fieldPath := "spec.platform.gcp.userTags"
	cases := []struct {
		name     string
		userTags []gcp.UserTag
		wantErr  bool
	}{
		{
			name:     "userTags not configured",
			userTags: []gcp.UserTag{},
			wantErr:  false,
		},
		{
			name: "userTags configured",
			userTags: []gcp.UserTag{
				{ParentID: "1234567890", Key: "key_2", Value: "value_2"},
				{ParentID: "test-project-123", Key: "key.gcp", Value: "value.3"},
				{ParentID: "1234567890", Key: "keY", Value: "value"},
				{ParentID: "test-project-123", Key: "thisisalongkeywithinlimitof63_characters-whichisallowedfortags", Value: "value"},
				{ParentID: "1234567890", Key: "KEY4", Value: "hisisavaluewithin-63characters_{[(.@%=+: ,*#&)]}forgcptagvalue"},
				{ParentID: "test-project-123", Key: "key1", Value: "value1"},
			},
			wantErr: false,
		},
		{
			name: "userTags configured is more than max limit",
			userTags: []gcp.UserTag{
				{ParentID: "1234567890", Key: "key29", Value: "value29"},
				{ParentID: "test-project-123", Key: "key33", Value: "value33"},
				{ParentID: "1234567890", Key: "key39", Value: "value39"},
				{ParentID: "test-project-123", Key: "key43", Value: "value43"},
				{ParentID: "1234567890", Key: "key5", Value: "value5"},
				{ParentID: "test-project-123", Key: "key6", Value: "value6"},
				{ParentID: "1234567890", Key: "key14", Value: "value14"},
				{ParentID: "test-project-123", Key: "key25", Value: "value25"},
				{ParentID: "1234567890", Key: "key20", Value: "value20"},
				{ParentID: "test-project-123", Key: "key24", Value: "value24"},
				{ParentID: "1234567890", Key: "key40", Value: "value40"},
				{ParentID: "test-project-123", Key: "key46", Value: "value46"},
				{ParentID: "1234567890", Key: "key1", Value: "value1"},
				{ParentID: "test-project-123", Key: "key2", Value: "value2"},
				{ParentID: "1234567890", Key: "key4", Value: "value4"},
				{ParentID: "test-project-123", Key: "key10", Value: "value10"},
				{ParentID: "1234567890", Key: "key51", Value: "value51"},
				{ParentID: "test-project-123", Key: "key8", Value: "value8"},
				{ParentID: "1234567890", Key: "key13", Value: "value13"},
				{ParentID: "test-project-123", Key: "key44", Value: "value44"},
				{ParentID: "1234567890", Key: "key48", Value: "value48"},
				{ParentID: "test-project-123", Key: "key9", Value: "value9"},
				{ParentID: "1234567890", Key: "key17", Value: "value17"},
				{ParentID: "test-project-123", Key: "key18", Value: "value18"},
				{ParentID: "1234567890", Key: "key30", Value: "value30"},
				{ParentID: "test-project-123", Key: "key36", Value: "value36"},
				{ParentID: "1234567890", Key: "key49", Value: "value49"},
				{ParentID: "test-project-123", Key: "key7", Value: "value7"},
				{ParentID: "1234567890", Key: "key15", Value: "value15"},
				{ParentID: "test-project-123", Key: "key22", Value: "value22"},
				{ParentID: "1234567890", Key: "key34", Value: "value34"},
				{ParentID: "test-project-123", Key: "key37", Value: "value37"},
				{ParentID: "1234567890", Key: "key38", Value: "value38"},
				{ParentID: "test-project-123", Key: "key47", Value: "value47"},
				{ParentID: "1234567890", Key: "key12", Value: "value12"},
				{ParentID: "test-project-123", Key: "key16", Value: "value16"},
				{ParentID: "1234567890", Key: "key23", Value: "value23"},
				{ParentID: "test-project-123", Key: "key28", Value: "value28"},
				{ParentID: "1234567890", Key: "key50", Value: "value50"},
				{ParentID: "test-project-123", Key: "key21", Value: "value21"},
				{ParentID: "1234567890", Key: "key26", Value: "value26"},
				{ParentID: "test-project-123", Key: "key35", Value: "value35"},
				{ParentID: "1234567890", Key: "key42", Value: "value42"},
				{ParentID: "test-project-123", Key: "key31", Value: "value31"},
				{ParentID: "1234567890", Key: "key32", Value: "value32"},
				{ParentID: "test-project-123", Key: "key41", Value: "value41"},
				{ParentID: "1234567890", Key: "key45", Value: "value45"},
				{ParentID: "test-project-123", Key: "key3", Value: "value3"},
				{ParentID: "1234567890", Key: "key11", Value: "value11"},
				{ParentID: "test-project-123", Key: "key19", Value: "value19"},
				{ParentID: "1234567890", Key: "key27", Value: "value27"},
			},
			wantErr: true,
		},
		{
			name:     "userTags contains key starting with a special character",
			userTags: []gcp.UserTag{{ParentID: "1234567890", Key: "_key", Value: "1value"}},
			wantErr:  true,
		},
		{
			name:     "userTags contains key ending with a special character",
			userTags: []gcp.UserTag{{ParentID: "1234567890", Key: "key@", Value: "1value"}},
			wantErr:  true,
		},
		{
			name:     "userTags contains empty key",
			userTags: []gcp.UserTag{{ParentID: "1234567890", Key: "", Value: "value"}},
			wantErr:  true,
		},
		{
			name: "userTags contains key length greater than 63",
			userTags: []gcp.UserTag{
				{
					ParentID: "1234567890",
					Key:      "thisisalongkeyforlimitof63_characters-whichisnotallowedfortagkey",
					Value:    "value",
				},
			},
			wantErr: true,
		},
		{
			name:     "userTags contains key with invalid character",
			userTags: []gcp.UserTag{{ParentID: "1234567890", Key: "key/test", Value: "value"}},
			wantErr:  true,
		},
		{
			name: "userTags contains value length greater than 63",
			userTags: []gcp.UserTag{
				{
					ParentID: "1234567890",
					Key:      "key",
					Value:    "hisisavaluewith-63characters_{[(.@%=+: ,*#&)]}allowedforgcptagvalue",
				},
			},
			wantErr: true,
		},
		{
			name:     "userTags contains empty value",
			userTags: []gcp.UserTag{{ParentID: "1234567890", Key: "key", Value: ""}},
			wantErr:  true,
		},
		{
			name:     "userTags contains value with invalid character",
			userTags: []gcp.UserTag{{ParentID: "1234567890", Key: "key", Value: "value*^%"}},
			wantErr:  true,
		},
		{
			name:     "userTags contains empty ParentID",
			userTags: []gcp.UserTag{{Key: "key", Value: "value*^%"}},
			wantErr:  true,
		},
		{
			name:     "userTags contains ParentID configured with invalid OrganizationID",
			userTags: []gcp.UserTag{{ParentID: "00001234567890", Key: "key", Value: "value"}},
			wantErr:  true,
		},
		{
			name:     "userTags contains ParentID configured with invalid ProjectID",
			userTags: []gcp.UserTag{{ParentID: "test-project-123-", Key: "key", Value: "value"}},
			wantErr:  true,
		},
		{
			name: "userTags contains ParentID configured with invalid OrganizationID length",
			userTags: []gcp.UserTag{
				{
					ParentID: "123456789012345678901234567890123",
					Key:      "key",
					Value:    "value",
				},
			},
			wantErr: true,
		},
		{
			name: "userTags contains ParentID configured with invalid ProjectID length",
			userTags: []gcp.UserTag{
				{
					ParentID: "test-project-123-test-project-123-test-project-123-test-project-123",
					Key:      "key",
					Value:    "value",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUserTags(tt.userTags, field.NewPath(fieldPath))
			if (len(err) > 0) != tt.wantErr {
				t.Errorf("unexpected error, err: %v", err)
			}
		})
	}
}
