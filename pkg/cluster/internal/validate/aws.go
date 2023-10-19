/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package validate

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/apparentlymart/go-cidr/cidr"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"golang.org/x/exp/slices"
	"sigs.k8s.io/kind/pkg/commons"
	"sigs.k8s.io/kind/pkg/errors"
)

const (
	cidrSizeMax = 65536
	cidrSizeMin = 16
)

var AWSVolumes = []string{"io1", "io2", "gp2", "gp3", "sc1", "st1", "standard", "sbp1", "sbg1"}
var isAWSNodeImage = regexp.MustCompile(`^ami-\w+$`).MatchString
var AWSNodeImageFormat = "ami-[IMAGE_ID]"

func validateAWS(spec commons.Spec, providerSecrets map[string]string) error {
	var err error
	var ctx = context.TODO()

	cfg, err := commons.AWSGetConfig(ctx, providerSecrets, spec.Region)
	if err != nil {
		return err
	}

	regions, err := getAWSRegions(cfg)
	if err != nil {
		return err
	}
	if !commons.Contains(regions, spec.Region) {
		return errors.New("spec.region: " + spec.Region + " region does not exist")
	}

	azs, err := getAWSAzs(ctx, cfg, spec.Region)
	if err != nil {
		return err
	}

	if (spec.StorageClass != commons.StorageClass{}) {
		if err = validateAWSStorageClass(spec.StorageClass, spec.WorkerNodes); err != nil {
			return errors.Wrap(err, "spec.storageclass: Invalid value")
		}
	}

	if !reflect.ValueOf(spec.Networks).IsZero() {
		if err = validateAWSNetwork(ctx, cfg, spec); err != nil {
			return errors.Wrap(err, "spec.networks: Invalid value")
		}
	}

	for i, dr := range spec.DockerRegistries {
		if dr.Type != "ecr" && dr.Type != "generic" {
			return errors.New("spec.docker_registries[" + strconv.Itoa(i) + "]: Invalid value: \"type\": only 'ecr' or 'generic' are supported in aws clusters")
		}
	}

	if !spec.ControlPlane.Managed {
		if spec.ControlPlane.NodeImage != "" {
			if !isAWSNodeImage(spec.ControlPlane.NodeImage) {
				return errors.New("spec.control_plane: Invalid value: \"node_image\": must have the format " + AWSNodeImageFormat)
			}
		}
		if err := validateVolumeType(spec.ControlPlane.RootVolume.Type, AWSVolumes); err != nil {
			return errors.Wrap(err, "spec.control_plane.root_volume: Invalid value: \"type\"")
		}

		for i, ev := range spec.ControlPlane.ExtraVolumes {
			if ev.DeviceName == "" {
				return errors.New("spec.control_plane.extra_volumes[" + strconv.Itoa(i) + "]: Invalid value: \"device_name\": is required")
			}
			if err := validateVolumeType(ev.Type, AWSVolumes); err != nil {
				return errors.Wrap(err, "spec.control_plane.extra_volumes["+strconv.Itoa(i)+"]: Invalid value: \"type\"")
			}
			for j, ev2 := range spec.ControlPlane.ExtraVolumes {
				if i != j {
					if ev.DeviceName == ev2.DeviceName {
						return errors.New("spec.control_plane.extra_volumes[" + strconv.Itoa(i) + "]: Invalid value: \"device_name\": is duplicated")
					}
				}
			}
		}
	}

	for _, wn := range spec.WorkerNodes {
		if wn.NodeImage != "" {
			if !isAWSNodeImage(wn.NodeImage) {
				return errors.New("spec.worker_nodes." + wn.Name + ": \"node_image\": must have the format " + AWSNodeImageFormat)
			}
		}
		if wn.AZ != "" {
			if len(azs) > 0 {
				if !commons.Contains(azs, wn.AZ) {
					return errors.New(wn.AZ + " does not exist in this region, azs: " + fmt.Sprint(azs))
				}
			}
		}
		if err := validateVolumeType(wn.RootVolume.Type, AWSVolumes); err != nil {
			return errors.Wrap(err, "spec.worker_nodes."+wn.Name+".root_volume: Invalid value: \"type\"")
		}
		for i, ev := range wn.ExtraVolumes {
			if ev.DeviceName == "" {
				return errors.New("spec.worker_nodes." + wn.Name + ".extra_volumes[" + strconv.Itoa(i) + "]: Invalid value: \"device_name\": is required")
			}
			if err := validateVolumeType(ev.Type, AWSVolumes); err != nil {
				return errors.Wrap(err, "spec.worker_nodes."+wn.Name+".extra_volumes["+strconv.Itoa(i)+"]: Invalid value: \"type\"")
			}
			for j, ev2 := range spec.ControlPlane.ExtraVolumes {
				if i != j {
					if ev.DeviceName == ev2.DeviceName {
						return errors.New("spec.worker_nodes." + wn.Name + ".extra_volumes[" + strconv.Itoa(i) + "]: Invalid value: \"device_name\": is duplicated")
					}
				}
			}
		}
	}

	return nil
}

func validateAWSNetwork(ctx context.Context, cfg aws.Config, spec commons.Spec) error {
	var err error
	if spec.Networks.PodsCidrBlock != "" {
		if spec.ControlPlane.Managed {
			if err = validateAWSPodsNetwork(spec.Networks.PodsCidrBlock); err != nil {
				return err
			}
		}
	} else {
		if len(spec.Networks.PodsSubnets) > 0 {
			return errors.New("\"pods_cidr\": is required when \"pods_subnets\" is set")
		}
	}
	if spec.Networks.VPCID != "" {
		vpcs, _ := getAWSVPCs(cfg)
		if len(vpcs) > 0 && !commons.Contains(vpcs, spec.Networks.VPCID) {
			return errors.New("\"vpc_id\": " + spec.Networks.VPCID + " does not exist")
		}
		if len(spec.Networks.Subnets) == 0 {
			return errors.New("\"subnets\": are required when \"vpc_id\" is set")
		} else {
			subnets, _ := getAWSSubnets(spec.Networks.VPCID, cfg)
			if len(subnets) > 0 {
				for _, subnet := range spec.Networks.Subnets {
					if !commons.Contains(subnets, subnet.SubnetId) {
						return errors.New("\"subnets\": " + subnet.SubnetId + " does not belong to vpc with id: " + spec.Networks.VPCID)
					}
				}
			}
		}
	} else {
		if len(spec.Networks.Subnets) > 0 {
			return errors.New("\"vpc_id\": is required when \"subnets\" is set")
		}
		if len(spec.Networks.PodsSubnets) > 0 {
			return errors.New("\"vpc_id\": is required when \"pods_subnets\" is set")
		}
	}
	if len(spec.Networks.Subnets) > 0 {
		for _, s := range spec.Networks.Subnets {
			if s.SubnetId == "" {
				return errors.New("\"subnet_id\": is required")
			}
		}
		if err = validateAWSAZs(ctx, cfg, spec); err != nil {
			return err
		}
	}
	return nil
}

func validateAWSPodsNetwork(podsNetwork string) error {
	// Minimum cidr range: 100.64.0.0/10
	validRange1 := net.IPNet{
		IP:   net.ParseIP("100.64.0.0"),
		Mask: net.IPv4Mask(255, 192, 0, 0),
	}
	// Maximum cidr range: 198.19.0.0/16
	validRange2 := net.IPNet{
		IP:   net.ParseIP("198.19.0.0"),
		Mask: net.IPv4Mask(255, 255, 0, 0),
	}

	_, ipv4Net, err := net.ParseCIDR(podsNetwork)
	if err != nil {
		return errors.New("\"pods_cidr\": CIDR block must be a valid IPv4 CIDR block")
	}

	cidrSize := cidr.AddressCount(ipv4Net)
	if cidrSize > cidrSizeMax || cidrSize < cidrSizeMin {
		return errors.New("\"pods_cidr\": CIDR block sizes must be between a /16 and /28 netmask")
	}

	start, end := cidr.AddressRange(ipv4Net)
	if (!validRange1.Contains(start) || !validRange1.Contains(end)) && (!validRange2.Contains(start) || !validRange2.Contains(end)) {
		return errors.New("\"pods_cidr\": CIDR block must be between " + validRange1.String() + " and " + validRange2.String())
	}
	return nil
}

func getAWSRegions(config aws.Config) ([]string, error) {
	regions := []string{}

	// Use a default region to authenticate
	config.Region = *aws.String("eu-west-1")

	client := ec2.NewFromConfig(config)

	// Describe regions
	describeRegionsOpts := &ec2.DescribeRegionsInput{}
	output, err := client.DescribeRegions(context.Background(), describeRegionsOpts)
	if err != nil {
		return nil, err
	}

	// Extract region names
	for _, region := range output.Regions {
		regions = append(regions, *region.RegionName)
	}

	return regions, nil
}

func getAWSVPCs(config aws.Config) ([]string, error) {
	vpcs := []string{}

	client := ec2.NewFromConfig(config)
	DescribeVpcOpts := &ec2.DescribeVpcsInput{}
	output, err := client.DescribeVpcs(context.Background(), DescribeVpcOpts)
	if err != nil {
		return []string{}, err
	}
	for _, vpc := range output.Vpcs {
		vpcs = append(vpcs, *vpc.VpcId)
	}
	return vpcs, nil
}

func getAWSSubnets(vpcId string, config aws.Config) ([]string, error) {
	subnets := []string{}

	client := ec2.NewFromConfig(config)
	vpc_id_filterName := "vpc-id"
	DescribeSubnetOpts := &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{
				Name:   &vpc_id_filterName,
				Values: []string{vpcId},
			},
		},
	}
	output, err := client.DescribeSubnets(context.Background(), DescribeSubnetOpts)
	if err != nil {
		return []string{}, err
	}
	for _, subnet := range output.Subnets {
		subnets = append(subnets, *subnet.SubnetId)
	}
	return subnets, nil
}

func validateAWSStorageClass(sc commons.StorageClass, wn commons.WorkerNodes) error {
	var err error
	var isKeyValid = regexp.MustCompile(`^arn:aws:kms:[a-zA-Z0-9-]+:\d{12}:key/[\w-]+$`).MatchString
	var AWSFSTypes = []string{"xfs", "ext3", "ext4", "ext2"}
	var AWSSCFields = []string{"Type", "FsType", "Labels", "AllowAutoIOPSPerGBIncrease", "BlockExpress", "BlockSize", "Iops", "IopsPerGB", "Encrypted", "KmsKeyId", "Throughput"}
	var AWSSCYamlFields = []string{"type", "fsType", "Labels", "allowAutoIOPSPerGBIncrease", "blockExpress", "blockSize", "iops", "iopsPerGB", "encrypted", "kmsKeyId", "throughput"}
	var typesSupportedForIOPS = []string{"io1", "io2", "gp3"}
	var iopsValue string
	var iopsKey string

	// Validate fields
	fields := getFieldNames(sc.Parameters)
	for _, f := range fields {
		if !commons.Contains(AWSSCFields, f) {
			return errors.New("\"parameters\": unsupported " + f + ", supported fields: " + fmt.Sprint(strings.Join(AWSSCYamlFields, ", ")))
		}
	}
	// Validate class
	if sc.Class != "" && sc.Parameters != (commons.SCParameters{}) {
		return errors.New("\"class\": cannot be set when \"parameters\" is set")
	}
	// Validate type
	if sc.Parameters.Type != "" && !commons.Contains(AWSVolumes, sc.Parameters.Type) {
		return errors.New("\"type\": unsupported " + sc.Parameters.Type + ", supported types: " + fmt.Sprint(strings.Join(AWSVolumes, ", ")))
	}
	// Validate encryptionKey format
	if sc.EncryptionKey != "" {
		if sc.Parameters != (commons.SCParameters{}) {
			return errors.New("\"encryptionKey\": cannot be set when \"parameters\" is set")
		}
		if !isKeyValid(sc.EncryptionKey) {
			return errors.New("\"encryptionKey\": it must have the format arn:aws:kms:[REGION]:[ACCOUNT_ID]:key/[KEY_ID]")
		}
	}
	// Validate diskEncryptionSetID format
	if sc.Parameters.KmsKeyId != "" {
		if !isKeyValid(sc.Parameters.KmsKeyId) {
			return errors.New("\"kmsKeyId\": it must have the format arn:aws:kms:[REGION]:[ACCOUNT_ID]:key/[KEY_ID]")
		}
		if sc.Parameters.Encrypted != "true" {
			return errors.New("\"kmsKeyId\": cannot be set when \"parameters.encrypted\" is not set to true")
		}
	}
	// Validate fsType
	if sc.Parameters.FsType != "" && !commons.Contains(AWSFSTypes, sc.Parameters.FsType) {
		return errors.New("\fsType\": unsupported " + sc.Parameters.FsType + ", supported types: " + fmt.Sprint(strings.Join(AWSFSTypes, ", ")))
	}
	// Validate iops
	if sc.Parameters.Iops != "" {
		iopsValue = sc.Parameters.Iops
		iopsKey = "iops"
	}
	if sc.Parameters.IopsPerGB != "" {
		iopsValue = sc.Parameters.IopsPerGB
		iopsKey = "iopsPerGB"
	}
	if iopsValue != "" && sc.Parameters.Type != "" && !slices.Contains(typesSupportedForIOPS, sc.Parameters.Type) {
		return errors.New(iopsKey + " only can be specified for " + fmt.Sprint(strings.Join(typesSupportedForIOPS, ", ")) + " types")
	}
	if iopsValue != "" {
		iops, err := strconv.Atoi(iopsValue)
		if err != nil {
			return errors.New("invalid " + iopsKey + " parameter. It must be a number in string format")
		}
		if (sc.Class == "standard" && sc.Parameters.Type == "") || sc.Parameters.Type == "gp3" {
			if iops < 3000 || iops > 16000 {
				return errors.New("invalid " + iopsKey + " parameter. It must be greater than 3000 and lower than 16000 for gp3 type")
			}
		}
		if (sc.Class == "premium" && sc.Parameters.Type == "") || sc.Parameters.Type == "io1" || sc.Parameters.Type == "io2" {
			if iops < 16000 || iops > 64000 {
				return errors.New("invalid " + iopsKey + " parameter. It must be greater than 16000 and lower than 64000 for io1 and io2 types")
			}
		}
	}
	// Validate labels
	if sc.Parameters.Labels != "" {
		if err = validateLabel(sc.Parameters.Labels); err != nil {
			return errors.Wrap(err, "invalid labels")
		}
	}
	return nil
}

func validateAWSAZs(ctx context.Context, cfg aws.Config, spec commons.Spec) error {
	var err error
	var azs []string

	svc := ec2.NewFromConfig(cfg)
	if len(spec.Networks.Subnets) > 0 {
		azs, err = commons.AWSGetPrivateAZs(ctx, svc, spec.Networks.Subnets)
		if err != nil {
			return err
		}
		if len(azs) < 3 {
			return errors.New("insufficient Availability Zones in region " + spec.Region + ". Please add at least 3 private subnets in different Availability Zones")
		}
	} else {
		azs, err = commons.AWSGetAZs(ctx, svc)
		if err != nil {
			return err
		}
		if len(azs) < 3 {
			return errors.New("insufficient Availability Zones in region " + spec.Region + ". Must have at least 3")
		}
	}

	for _, node := range spec.WorkerNodes {
		if node.ZoneDistribution == "unbalanced" && node.AZ != "" {
			if !slices.Contains(azs, node.AZ) {
				return errors.New("worker node " + node.Name + " whose AZ is defined in " + node.AZ + " must match with the AZs associated to the defined subnets in descriptor")
			}
		}
	}

	return nil
}

func getAWSAzs(ctx context.Context, cfg aws.Config, region string) ([]string, error) {
	var azs []string
	svc := ec2.NewFromConfig(cfg)
	result, err := svc.DescribeAvailabilityZones(ctx, &ec2.DescribeAvailabilityZonesInput{})
	if err != nil {
		return nil, err
	}
	for _, az := range result.AvailabilityZones {
		if *az.RegionName == region {
			azs = append(azs, *az.ZoneName)
		}
	}
	return azs, nil
}
