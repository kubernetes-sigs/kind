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
	"sigs.k8s.io/kind/pkg/commons"
)

type ValidateParams struct {
	KeosCluster   commons.KeosCluster
	SecretsPath   string
	VaultPassword string
}

func Cluster(params *ValidateParams) (commons.ClusterCredentials, error) {
	var err error
	var creds commons.ClusterCredentials

	creds, err = validateCredentials(*params)
	if err != nil {
		return commons.ClusterCredentials{}, err
	}

	if err := validateCommon(params.KeosCluster.Spec); err != nil {
		return commons.ClusterCredentials{}, err
	}

	switch params.KeosCluster.Spec.InfraProvider {
	case "aws":
		err = validateAWS(params.KeosCluster.Spec, creds.ProviderCredentials)
	case "gcp":
		err = validateGCP(params.KeosCluster.Spec, creds.ProviderCredentials)
	case "azure":
		err = validateAzure(params.KeosCluster.Spec, creds.ProviderCredentials, params.KeosCluster.Metadata.Name)
	}
	if err != nil {
		return commons.ClusterCredentials{}, err
	}

	return creds, nil
}
