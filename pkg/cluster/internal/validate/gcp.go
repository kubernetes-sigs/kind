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
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
	"sigs.k8s.io/kind/pkg/commons"
	"sigs.k8s.io/kind/pkg/errors"
)

var GCPVolumes = []string{"pd-balanced", "pd-ssd", "pd-standard", "pd-extreme"}
var GCPFSTypes = []string{"xfs", "ext3", "ext4", "ext2"}
var GCPSCFields = []string{"Type", "FsType", "Labels", "ProvisionedIopsOnCreate", "ProvisionedThroughputOnCreate", "ReplicationType"}

var isGCPNodeImage = regexp.MustCompile(`^projects/[\w-]+/global/images/[\w-]+$`).MatchString
var GCPNodeImageFormat = "projects/[PROJECT_ID]/global/images/[IMAGE_NAME]"

func validateGCP(spec commons.Spec) error {
	var err error

	if (spec.StorageClass != commons.StorageClass{}) {
		if err = validateGCPStorageClass(spec); err != nil {
			return errors.Wrap(err, "invalid storage class")
		}
	}

	if !reflect.ValueOf(spec.Networks).IsZero() {
		if err = validateGCPNetwork(spec.Networks); err != nil {
			return errors.Wrap(err, "invalid network")
		}
	}

	if !spec.ControlPlane.Managed {
		if spec.ControlPlane.NodeImage == "" || !isGCPNodeImage(spec.ControlPlane.NodeImage) {
			return errors.New("incorrect control plane node image. It must be present and have the format " + GCPNodeImageFormat)
		}
		if err = validateGCPVolumes(spec.ControlPlane.RootVolume, spec.ControlPlane.ExtraVolumes); err != nil {
			return errors.Wrap(err, "invalid control plane volumes")
		}
	}

	for _, wn := range spec.WorkerNodes {
		if wn.NodeImage == "" || !isGCPNodeImage(wn.NodeImage) {
			return errors.New("incorrect worker " + wn.Name + " node image. It must be present and have the format " + GCPNodeImageFormat)
		}
		if err = validateGCPVolumes(wn.RootVolume, wn.ExtraVolumes); err != nil {
			return errors.Wrap(err, "invalid worker node volumes")
		}
	}

	return nil
}

func validateGCPStorageClass(spec commons.Spec) error {
	var err error
	var isKeyValid = regexp.MustCompile(`^projects/[a-zA-Z0-9-]+/locations/[a-zA-Z0-9-]+/keyRings/[a-zA-Z0-9-]+/cryptoKeys/[a-zA-Z0-9-]+$`).MatchString
	var sc = spec.StorageClass

	// Validate fields
	fields := getFieldNames(sc.Parameters)
	for _, f := range fields {
		if !commons.Contains(GCPSCFields, f) {
			return errors.New("field " + strcase.ToLowerCamel(f) + " is not supported in storage class")
		}
	}

	// Validate encryptionKey format
	if sc.EncryptionKey != "" {
		if !isKeyValid(sc.EncryptionKey) {
			return errors.New("incorrect encryptionKey format. It must have the format projects/[PROJECT_ID]/locations/[REGION]/keyRings/[RING_NAME]/cryptoKeys/[KEY_NAME]")
		}
	}
	// Validate disk-encryption-kms-key format
	if sc.Parameters.DiskEncryptionKmsKey != "" {
		if !isKeyValid(sc.Parameters.DiskEncryptionKmsKey) {
			return errors.New("incorrect disk-encryption-kms-key format. It must have the format projects/[PROJECT_ID]/locations/[REGION]/keyRings/[RING_NAME]/cryptoKeys/[KEY_NAME]")
		}
	}
	// Validate type
	if sc.Parameters.Type != "" && !commons.Contains(GCPVolumes, sc.Parameters.Type) {
		return errors.New("unsupported type: " + sc.Parameters.Type)
	}
	// Validate fsType
	if sc.Parameters.FsType != "" && !commons.Contains(GCPFSTypes, sc.Parameters.FsType) {
		return errors.New("unsupported fsType: " + sc.Parameters.FsType + ". Supported types: " + fmt.Sprint(strings.Join(GCPFSTypes, ", ")))
	}

	if spec.ControlPlane.Managed {
		version, _ := strconv.ParseFloat(regexp.MustCompile(".[0-9]+$").Split(strings.ReplaceAll(spec.K8SVersion, "v", ""), -1)[0], 64)
		if sc.Parameters.Type == "pd-extreme" && version < 1.26 {
			return errors.New("pd-extreme is only supported in GKE 1.26 or later")
		}
	}
	// Validate provisioned-iops-on-create
	if sc.Parameters.ProvisionedIopsOnCreate != "" {
		if sc.Parameters.Type != "pd-extreme" {
			return errors.New("provisioned-iops-on-create is only supported for pd-extreme")
		}
		if _, err = strconv.Atoi(sc.Parameters.ProvisionedIopsOnCreate); err != nil {
			return errors.New("provisioned-iops-on-create must be an integer")
		}
	}
	// Validate replication-type
	if sc.Parameters.ReplicationType != "" && !regexp.MustCompile(`^(none|regional-pd)$`).MatchString(sc.Parameters.ReplicationType) {
		return errors.New("incorrect replication-type. Supported values are 'none' or 'regional-pd'")
	}
	// Validate labels
	if sc.Parameters.Labels != "" {
		if err = validateLabel(sc.Parameters.Labels); err != nil {
			return errors.Wrap(err, "invalid labels")
		}
	}
	return nil
}

func validateGCPNetwork(network commons.Networks) error {
	if network.VPCID == "" {
		return errors.New("vpc_id is required")
	}
	if len(network.Subnets) > 0 {
		if len(network.Subnets) > 1 {
			return errors.New("only one subnet is supported")
		}
		if network.Subnets[0].SubnetId == "" {
			return errors.New("subnet_id is required")
		}
	}
	return nil
}

func validateGCPVolumes(rootVol commons.RootVolume, extraVols []commons.ExtraVolume) error {
	var err error
	if err = validateVolumeType(rootVol.Type, GCPVolumes); err != nil {
		return errors.Wrap(err, "invalid root volume")
	}
	for _, v := range extraVols {
		if err = validateVolumeType(v.Type, GCPVolumes); err != nil {
			return errors.Wrap(err, "invalid extra volume")
		}
	}
	return nil
}
