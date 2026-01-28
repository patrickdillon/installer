// Package aws generates Machine objects for aws.
package aws

import (
	"fmt"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	capa "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	capi "sigs.k8s.io/cluster-api/api/core/v1beta1" //nolint:staticcheck //CORS-3563

	icaws "github.com/openshift/installer/pkg/asset/installconfig/aws"
	"github.com/openshift/installer/pkg/types"
	"github.com/openshift/installer/pkg/types/aws"
)

// ClusterAPIMachineSetInput defines inputs for generating CAPI MachineSets.
type ClusterAPIMachineSetInput struct {
	ClusterID                string
	InstallConfigPlatformAWS *aws.Platform
	Subnets                  icaws.SubnetsByZone
	Zones                    icaws.Zones
	PublicSubnet             bool
	Pool                     *types.MachinePool
	Role                     string
	UserDataSecret           string
}

// ClusterAPIMachineSetOutput contains the generated CAPI resources.
type ClusterAPIMachineSetOutput struct {
	MachineTemplates []capa.AWSMachineTemplate
	MachineSets      []capi.MachineSet
}

// ClusterAPIMachineSets returns CAPI MachineSet and AWSMachineTemplate resources.
// This mirrors the MAPI MachineSets() function but produces CAPI-native types.
func ClusterAPIMachineSets(in *ClusterAPIMachineSetInput) (*ClusterAPIMachineSetOutput, error) {
	if poolPlatform := in.Pool.Platform.Name(); poolPlatform != aws.Name {
		return nil, fmt.Errorf("non-AWS machine-pool: %q", poolPlatform)
	}
	mpool := in.Pool.Platform.AWS
	azs := mpool.Zones

	total := int64(0)
	if in.Pool.Replicas != nil {
		total = *in.Pool.Replicas
	}
	numOfAZs := int64(len(azs))

	var templates []capa.AWSMachineTemplate
	var machineSets []capi.MachineSet

	imds := capa.HTTPTokensStateOptional
	if mpool.EC2Metadata.Authentication == "Required" {
		imds = capa.HTTPTokensStateRequired
	}

	instanceProfile := mpool.IAMProfile
	if len(instanceProfile) == 0 {
		instanceProfile = fmt.Sprintf("%s-worker-profile", in.ClusterID)
	}

	tags, err := CapaTagsFromUserTags(in.ClusterID, in.InstallConfigPlatformAWS.UserTags)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create CAPA tags from user tags")
	}

	for idx, az := range mpool.Zones {
		replicas := int32(total / numOfAZs)
		if int64(idx) < total%numOfAZs {
			replicas++
		}

		nodeLabels := make(map[string]string, 3)
		nodeTaints := []corev1.Taint{}
		instanceType := mpool.InstanceType
		publicSubnet := in.PublicSubnet
		subnetRef := &capa.AWSResourceReference{}

		if len(in.Subnets) > 0 {
			subnet, ok := in.Subnets[az]
			if !ok {
				return nil, errors.Errorf("no subnet for zone %s", az)
			}
			publicSubnet = subnet.Public
			subnetRef.ID = ptr.To(subnet.ID)
		} else {
			subnetInternetScope := "private"
			if publicSubnet {
				subnetInternetScope = "public"
			}
			subnetRef.Filters = []capa.Filter{
				{
					Name:   "tag:Name",
					Values: []string{fmt.Sprintf("%s-subnet-%s-%s", in.ClusterID, subnetInternetScope, az)},
				},
			}
		}

		if in.Pool.Name == types.MachinePoolEdgeRoleName {
			// edge pools not share same instance type and regular cluster workloads.
			// The instance type is selected based in the offerings for the location.
			// The labels and taints are set to prevent regular workloads.
			// https://github.com/openshift/enhancements/blob/master/enhancements/installer/aws-custom-edge-machineset-local-zones.md
			zone := in.Zones[az]
			if zone.PreferredInstanceType != "" {
				instanceType = zone.PreferredInstanceType
			}
			nodeLabels = map[string]string{
				"node-role.kubernetes.io/edge":          "",
				"machine.openshift.io/zone-type":        zone.Type,
				"machine.openshift.io/zone-group":       zone.GroupName,
				"machine.openshift.io/parent-zone-name": zone.ParentZoneName,
			}
			nodeTaints = append(nodeTaints, corev1.Taint{
				Key:    "node-role.kubernetes.io/edge",
				Effect: "NoSchedule",
			})
		}

		name := fmt.Sprintf("%s-%s-%s", in.ClusterID, in.Pool.Name, az)

		// Build AWSMachineTemplate for this zone
		templateSpec := capa.AWSMachineSpec{
			InstanceType:       instanceType,
			AMI:                capa.AMIReference{ID: ptr.To(mpool.AMIID)},
			SSHKeyName:         ptr.To(""),
			IAMInstanceProfile: instanceProfile,
			Subnet:             subnetRef,
			PublicIP:           ptr.To(publicSubnet),
			AdditionalTags:     tags,
			RootVolume: &capa.Volume{
				Size:      int64(mpool.EC2RootVolume.Size),
				Type:      capa.VolumeType(mpool.EC2RootVolume.Type),
				IOPS:      int64(mpool.EC2RootVolume.IOPS),
				Encrypted: ptr.To(true),
			},
			InstanceMetadataOptions: &capa.InstanceMetadataOptions{
				HTTPTokens:   imds,
				HTTPEndpoint: capa.InstanceMetadataEndpointStateEnabled,
			},
			UncompressedUserData: ptr.To(true),
			Ignition: &capa.Ignition{
				Version:     "3.2",
				StorageType: capa.IgnitionStorageTypeOptionUnencryptedUserData,
			},
		}

		if throughput := mpool.EC2RootVolume.Throughput; throughput != nil {
			templateSpec.RootVolume.Throughput = ptr.To(int64(*throughput))
		}

		if mpool.KMSKeyARN != "" {
			templateSpec.RootVolume.EncryptionKey = mpool.KMSKeyARN
		}

		// Handle additional security groups.
		for _, sg := range mpool.AdditionalSecurityGroupIDs {
			templateSpec.AdditionalSecurityGroups = append(
				templateSpec.AdditionalSecurityGroups,
				capa.AWSResourceReference{ID: ptr.To(sg)},
			)
		}

		if mpool.CPUOptions != nil {
			cpuOptions := capa.CPUOptions{}
			if mpool.CPUOptions.ConfidentialCompute != nil {
				cpuOptions.ConfidentialCompute = capa.AWSConfidentialComputePolicy(*mpool.CPUOptions.ConfidentialCompute)
			}
			templateSpec.CPUOptions = cpuOptions
		}

		template := capa.AWSMachineTemplate{
			TypeMeta: metav1.TypeMeta{
				APIVersion: capa.GroupVersion.String(),
				Kind:       "AWSMachineTemplate",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "openshift-cluster-api",
				Labels: map[string]string{
					"cluster.x-k8s.io/cluster-name": in.ClusterID,
				},
			},
			Spec: capa.AWSMachineTemplateSpec{
				Template: capa.AWSMachineTemplateResource{
					Spec: templateSpec,
				},
			},
		}
		templates = append(templates, template)

		// Build CAPI MachineSet referencing the template
		machineLabels := map[string]string{
			"cluster.x-k8s.io/cluster-name": in.ClusterID,
		}
		for k, v := range nodeLabels {
			machineLabels[k] = v
		}

		machineSet := capi.MachineSet{
			TypeMeta: metav1.TypeMeta{
				APIVersion: capi.GroupVersion.String(),
				Kind:       "MachineSet",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "openshift-cluster-api",
				Labels: map[string]string{
					"cluster.x-k8s.io/cluster-name": in.ClusterID,
				},
			},
			Spec: capi.MachineSetSpec{
				ClusterName: in.ClusterID,
				Replicas:    &replicas,
				Selector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"cluster.x-k8s.io/cluster-name":  in.ClusterID,
						"cluster.x-k8s.io/set-name":      name,
						"cluster.x-k8s.io/deployment":    in.Role,
						"topology.cluster.x-k8s.io/zone": az,
					},
				},
				Template: capi.MachineTemplateSpec{
					ObjectMeta: capi.ObjectMeta{
						Labels: map[string]string{
							"cluster.x-k8s.io/cluster-name":  in.ClusterID,
							"cluster.x-k8s.io/set-name":      name,
							"cluster.x-k8s.io/deployment":    in.Role,
							"topology.cluster.x-k8s.io/zone": az,
						},
					},
					Spec: capi.MachineSpec{
						ClusterName: in.ClusterID,
						Bootstrap: capi.Bootstrap{
							DataSecretName: ptr.To(in.UserDataSecret),
						},
						InfrastructureRef: corev1.ObjectReference{
							APIVersion: capa.GroupVersion.String(),
							Kind:       "AWSMachineTemplate",
							Name:       name,
							Namespace:  "openshift-cluster-api",
						},
						NodeDrainTimeout: &metav1.Duration{},
					},
				},
			},
		}

		// Apply node taints if any (for edge pools)
		if len(nodeTaints) > 0 {
			// Note: CAPI doesn't directly support taints on MachineSet.
			// Taints would typically be applied via MachineDeployment or node configuration.
			// For CAPI, we add labels that operators can use to apply taints.
			for k, v := range nodeLabels {
				machineSet.Spec.Template.ObjectMeta.Labels[k] = v
			}
		}

		machineSets = append(machineSets, machineSet)
	}

	return &ClusterAPIMachineSetOutput{
		MachineTemplates: templates,
		MachineSets:      machineSets,
	}, nil
}
