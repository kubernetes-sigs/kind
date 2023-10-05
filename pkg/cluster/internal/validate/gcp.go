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
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	b64 "encoding/base64"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
	"sigs.k8s.io/kind/pkg/commons"
	"sigs.k8s.io/kind/pkg/errors"
)

var GCPVolumes = []string{"pd-balanced", "pd-ssd", "pd-standard", "pd-extreme"}
var isGCPNodeImage = regexp.MustCompile(`^projects/[\w-]+/global/images/[\w-]+$`).MatchString
var GCPNodeImageFormat = "projects/[PROJECT_ID]/global/images/[IMAGE_NAME]"

func validateGCP(spec commons.Spec, providerSecrets map[string]string) error {
	var err error
	var isGKEVersion = regexp.MustCompile(`^v\d.\d{2}.\d{1,2}-gke.\d{3,4}$`).MatchString

	credentialsJson := getGCPCreds(providerSecrets)
	azs, err := getGoogleAZs(credentialsJson, spec.Region)
	if err != nil {
		return err
	}
	if (spec.StorageClass != commons.StorageClass{}) {
		if err = validateGCPStorageClass(spec); err != nil {
			return errors.Wrap(err, "spec.storageclass: Invalid value")
		}
	}

	if !reflect.ValueOf(spec.Networks).IsZero() {
		if err = validateGCPNetwork(spec.Networks, credentialsJson, spec.Region); err != nil {
			return errors.Wrap(err, "spec.networks: Invalid value")
		}
	}

	for i, dr := range spec.DockerRegistries {
		if dr.Type != "gar" && dr.Type != "gcr" && spec.ControlPlane.Managed {
			return errors.New("spec.docker_registries[" + strconv.Itoa(i) + "]: Invalid value: \"type\": only 'gar' and 'gcr' are supported in gcp managed clusters")
		}
		if dr.Type != "gar" && dr.Type != "gcr" && dr.Type != "generic" {
			return errors.New("spec.docker_registries[" + strconv.Itoa(i) + "]: Invalid value: \"type\": only 'gar', 'gcr' and 'generic' are supported in gcp unmanaged clusters")
		}
	}

	if spec.ControlPlane.Managed {
		if !isGKEVersion(spec.K8SVersion) {
			return errors.New("spec: Invalid value: \"k8s_version\": must have the format 'v1.27.3-gke-1400'")
		}
	} else {
		if spec.ControlPlane.NodeImage == "" || !isGCPNodeImage(spec.ControlPlane.NodeImage) {
			return errors.New("spec.control_plane: Invalid value: \"node_image\": is required and have the format " + GCPNodeImageFormat)
		}
		if err := validateVolumeType(spec.ControlPlane.RootVolume.Type, GCPVolumes); err != nil {
			return errors.Wrap(err, "spec.control_plane.root_volume: Invalid value: \"type\"")
		}
		for i, ev := range spec.ControlPlane.ExtraVolumes {
			if err := validateVolumeType(ev.Type, GCPVolumes); err != nil {
				return errors.Wrap(err, "spec.control_plane.extra_volumes["+strconv.Itoa(i)+"]: Invalid value: \"type\"")
			}
		}
		for _, wn := range spec.WorkerNodes {
			if wn.NodeImage == "" || !isGCPNodeImage(wn.NodeImage) {
				return errors.New("spec.worker_nodes." + wn.Name + ": \"node_image\": is required and have the format " + GCPNodeImageFormat)
			}
			if err := validateVolumeType(wn.RootVolume.Type, GCPVolumes); err != nil {
				return errors.Wrap(err, "spec.worker_nodes."+wn.Name+".root_volume: Invalid value: \"type\"")
			}
			for i, ev := range wn.ExtraVolumes {
				if err := validateVolumeType(ev.Type, GCPVolumes); err != nil {
					return errors.Wrap(err, "spec.worker_nodes."+wn.Name+".extra_volumes["+strconv.Itoa(i)+"]: Invalid value: \"type\"")
				}
			}
		}
	}

	for _, wn := range spec.WorkerNodes {
		if wn.AZ != "" {
			if len(azs) > 0 {
				if !commons.Contains(azs, wn.AZ) {
					return errors.New(wn.AZ + " does not exist in this region, azs: " + fmt.Sprint(azs))
				}
			}
		}
	}

	return nil
}

func validateGCPStorageClass(spec commons.Spec) error {
	var err error
	var isKeyValid = regexp.MustCompile(`^projects/[a-zA-Z0-9-]+/locations/[a-zA-Z0-9-]+/keyRings/[a-zA-Z0-9-]+/cryptoKeys/[a-zA-Z0-9-]+$`).MatchString
	var sc = spec.StorageClass
	var GCPFSTypes = []string{"xfs", "ext3", "ext4", "ext2"}
	var GCPSCFields = []string{"Type", "FsType", "Labels", "DiskEncryptionKmsKey", "ProvisionedIopsOnCreate", "ProvisionedThroughputOnCreate", "ReplicationType"}
	var GCPYamlFields = []string{"type", "fsType", "labels", "disk-encryption-kms-key", "provisioned-iops-on-create", "provisioned-throughput-on-create", "replication-type"}

	// Validate fields
	fields := getFieldNames(sc.Parameters)
	for _, f := range fields {
		if !commons.Contains(GCPSCFields, f) {
			return errors.New("\"parameters\": unsupported " + f + ", supported fields: " + fmt.Sprint(strings.Join(GCPYamlFields, ", ")))
		}
	}
	// Validate class
	if sc.Class != "" && sc.Parameters != (commons.SCParameters{}) {
		return errors.New("\"class\": cannot be set when \"parameters\" is set")
	}
	// Validate type
	if sc.Parameters.Type != "" && !commons.Contains(GCPVolumes, sc.Parameters.Type) {
		return errors.New("\"type\": unsupported " + sc.Parameters.Type + ", supported types: " + fmt.Sprint(strings.Join(GCPVolumes, ", ")))
	}
	// Validate encryptionKey format
	if sc.EncryptionKey != "" {
		if sc.Parameters != (commons.SCParameters{}) {
			return errors.New("\"encryptionKey\": cannot be set when \"parameters\" is set")
		}
		if !isKeyValid(sc.EncryptionKey) {
			return errors.New("\"encryptionKey\": it must have the format projects/[PROJECT_ID]/locations/[REGION]/keyRings/[RING_NAME]/cryptoKeys/[KEY_NAME]")
		}
	}
	// Validate disk-encryption-kms-key format
	if sc.Parameters.DiskEncryptionKmsKey != "" {
		if !isKeyValid(sc.Parameters.DiskEncryptionKmsKey) {
			return errors.New("\"disk-encryption-kms-key\": it must have the format projects/[PROJECT_ID]/locations/[REGION]/keyRings/[RING_NAME]/cryptoKeys/[KEY_NAME]")
		}
	}
	// Validate fsType
	if sc.Parameters.FsType != "" && !commons.Contains(GCPFSTypes, sc.Parameters.FsType) {
		return errors.New("\fsType\": unsupported " + sc.Parameters.FsType + ", supported types: " + fmt.Sprint(strings.Join(GCPFSTypes, ", ")))
	}

	if spec.ControlPlane.Managed {
		version, _ := strconv.ParseFloat(regexp.MustCompile(".[0-9]+$").Split(strings.ReplaceAll(spec.K8SVersion, "v", ""), -1)[0], 64)
		if sc.Parameters.Type == "pd-extreme" && version < 1.26 {
			return errors.New("\"pd-extreme\": is only supported in GKE 1.26 or later")
		}
	}
	// Validate provisioned-iops-on-create
	if sc.Parameters.ProvisionedIopsOnCreate != "" {
		if sc.Parameters.Type != "pd-extreme" {
			return errors.New("\"provisioned-iops-on-create\": is only supported for pd-extreme type")
		}
		if _, err = strconv.Atoi(sc.Parameters.ProvisionedIopsOnCreate); err != nil {
			return errors.New("\"provisioned-iops-on-create\": must be an integer")
		}
	}
	// Validate replication-type
	if sc.Parameters.ReplicationType != "" && !regexp.MustCompile(`^(none|regional-pd)$`).MatchString(sc.Parameters.ReplicationType) {
		return errors.New("\"replication-type\": supported values are 'none' or 'regional-pd'")
	}
	// Validate labels
	if sc.Parameters.Labels != "" {
		if err = validateLabel(sc.Parameters.Labels); err != nil {
			return errors.Wrap(err, "invalid labels")
		}
	}
	return nil
}

func validateGCPNetwork(network commons.Networks, credentialsJson string, region string) error {
	if network.VPCID != "" {
		vpcs, _ := getGoogleVPCs(credentialsJson)
		if len(vpcs) > 0 && !commons.Contains(vpcs, network.VPCID) {
			return errors.New("\"vpc_id\": " + network.VPCID + " does not exist")
		}
		if len(network.Subnets) != 1 {
			return errors.New("\"subnet\": when \"vpc_id\" is set, one subnet must be specified")
		}
		if network.Subnets[0].SubnetId == "" {
			return errors.New("\"subnet_id\": required")
		}
		subnets, _ := getGoogleSubnets(credentialsJson, region, network.VPCID)
		if !commons.Contains(subnets, network.Subnets[0].SubnetId) {
			return errors.New("\"subnets\": " + network.Subnets[0].SubnetId + " does not belong to vpc with id: " + network.VPCID)
		}
	} else {
		if len(network.Subnets) > 0 {
			return errors.New("\"vpc_id\": is required when \"subnets\" is set")
		}
	}
	if len(network.Subnets) > 0 {
		if len(network.Subnets) > 1 {
			return errors.New("\"subnet\": only one subnet is supported")
		}
		if network.Subnets[0].SubnetId == "" {
			return errors.New("\"subnet_id\": required")
		}
	}
	return nil
}

func getGoogleVPCs(credentialsJson string) ([]string, error) {
	var network_names []string
	var ctx = context.Background()

	gcpCreds := map[string]string{}
	err := json.Unmarshal([]byte(credentialsJson), &gcpCreds)
	if err != nil {
		return []string{}, err
	}

	cfg := option.WithCredentialsJSON([]byte(credentialsJson))
	computeService, err := compute.NewService(ctx, cfg)

	if err != nil {
		return []string{}, err
	}

	networks, err := computeService.Networks.List(string(gcpCreds["project_id"])).Do()
	if err != nil {
		return []string{}, err
	}

	for _, network := range networks.Items {
		network_names = append(network_names, network.Name)
	}

	return network_names, nil

}

func getGoogleSubnets(credentialsJson string, region string, vpcId string) ([]string, error) {
	var subnetwork_names []string
	var ctx = context.Background()

	gcpCreds := map[string]string{}
	err := json.Unmarshal([]byte(credentialsJson), &gcpCreds)
	if err != nil {
		return []string{}, err
	}

	cfg := option.WithCredentialsJSON([]byte(credentialsJson))
	computeService, err := compute.NewService(ctx, cfg)

	if err != nil {
		return []string{}, err
	}

	subnetworks, err := computeService.Subnetworks.List(string(gcpCreds["project_id"]), region).Do()
	if err != nil {
		return []string{}, err
	}

	for _, subnetwork := range subnetworks.Items {
		networkParts := strings.Split(subnetwork.Network, "/")
		networkId := networkParts[len(networkParts)-1]
		if networkId == vpcId {
			subnetwork_names = append(subnetwork_names, subnetwork.Name)
		}
	}

	return subnetwork_names, nil

}

func getGoogleAZs(credentialsJson string, region string) ([]string, error) {
	var zones_names []string
	var ctx = context.Background()

	gcpCreds := map[string]string{}
	err := json.Unmarshal([]byte(credentialsJson), &gcpCreds)
	if err != nil {
		return []string{}, err
	}

	cfg := option.WithCredentialsJSON([]byte(credentialsJson))
	computeService, err := compute.NewService(ctx, cfg)

	if err != nil {
		return []string{}, err
	}

	zones, err := computeService.Zones.List(string(gcpCreds["project_id"])).Filter("name=" + region + "*").Do()
	if err != nil {
		return []string{}, err
	}

	for _, zone := range zones.Items {
		zones_names = append(zones_names, zone.Name)
	}

	return zones_names, nil
}

func getGCPCreds(providerSecrets map[string]string) string {
	data := map[string]interface{}{
		"type":                        "service_account",
		"project_id":                  providerSecrets["ProjectID"],
		"private_key_id":              providerSecrets["PrivateKeyID"],
		"private_key":                 providerSecrets["PrivateKey"],
		"client_email":                providerSecrets["ClientEmail"],
		"client_id":                   providerSecrets["ClientID"],
		"auth_uri":                    "https://accounts.google.com/o/oauth2/auth",
		"token_uri":                   "https://accounts.google.com/o/oauth2/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url":        "https://www.googleapis.com/robot/v1/metadata/x509/" + url.QueryEscape(providerSecrets["ClientEmail"]),
	}
	jsonData, _ := json.Marshal(data)
	credentials := b64.StdEncoding.EncodeToString([]byte(jsonData))
	credentialsJson, _ := b64.StdEncoding.DecodeString(credentials)
	return string(credentialsJson)
}
