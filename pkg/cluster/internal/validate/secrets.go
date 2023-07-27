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
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/fatih/structs"
	"github.com/oleiade/reflections"
	"sigs.k8s.io/kind/pkg/commons"
	"sigs.k8s.io/kind/pkg/errors"
)

func validateCredentials(params ValidateParams) (commons.ClusterCredentials, error) {
	var secrets commons.Secrets
	var creds commons.ClusterCredentials

	// Get secrets file if exists
	_, err := os.Stat(params.SecretsPath)
	if err == nil {
		secretsFile, err := commons.GetSecretsFile(params.SecretsPath, params.VaultPassword)
		if err != nil {
			return commons.ClusterCredentials{}, err
		}
		secrets = secretsFile.Secrets
	}

	creds.ProviderCredentials, err = validateProviderCredentials(secrets, params)
	if err != nil {
		return commons.ClusterCredentials{}, err
	}

	creds.KeosRegistryCredentials, creds.DockerRegistriesCredentials, err = validateRegistryCredentials(secrets, params.KeosCluster.Spec)
	if err != nil {
		return commons.ClusterCredentials{}, err
	}

	creds.GithubToken, err = validateGithubToken(secrets, params.KeosCluster.Spec)
	if err != nil {
		return commons.ClusterCredentials{}, err
	}

	return creds, nil
}

func validateProviderCredentials(secrets interface{}, params ValidateParams) (map[string]string, error) {
	infraProvider := params.KeosCluster.Spec.InfraProvider
	credentialsProvider, err := reflections.GetField(secrets, strings.ToUpper(infraProvider))
	if err != nil || reflect.DeepEqual(credentialsProvider, reflect.Zero(reflect.TypeOf(credentialsProvider)).Interface()) {
		credentialsProvider, err = reflections.GetField(params.KeosCluster.Spec.Credentials, strings.ToUpper(infraProvider))
		if err != nil || reflect.DeepEqual(credentialsProvider, reflect.Zero(reflect.TypeOf(credentialsProvider)).Interface()) {
			return nil, errors.New("there is not " + infraProvider + " credentials in descriptor or secrets file")
		}
	} else {
		credentialsProvider, _ = reflections.GetField(credentialsProvider, "Credentials")

	}
	err = validateStruct(credentialsProvider)
	if err != nil {
		return nil, err
	}
	resultCredsMap := structs.Map(credentialsProvider)
	resultCreds := convertToMapStringString(resultCredsMap)
	return resultCreds, nil
}

func validateRegistryCredentials(secrets commons.Secrets, spec commons.Spec) (map[string]string, []map[string]interface{}, error) {
	var dockerRegistries []commons.DockerRegistryCredentials
	var resultKeosRegistry map[string]string
	var resultDockerRegistries = []map[string]interface{}{}

	if len(secrets.DockerRegistries) > 0 {
		dockerRegistries = secrets.DockerRegistries
	} else {
		dockerRegistries = spec.Credentials.DockerRegistries
	}

	keosCount := 0
	for i, dockerRegistry := range spec.DockerRegistries {
		// Check if there are more than one docker_registry with the same URL
		for j, dockerRegistry2 := range spec.DockerRegistries {
			if i != j && dockerRegistry.URL == dockerRegistry2.URL {
				return nil, nil, errors.New("there is more than one docker_registry with the same URL: " + dockerRegistry.URL)
			}
		}
		if dockerRegistry.AuthRequired {
			existCredentials := false
			for l, dockerRegistryCredential := range dockerRegistries {
				// Check if there are more than one credential for the same registry
				for k, dockerRegistryCredential2 := range dockerRegistries {
					if l != k && dockerRegistryCredential.URL == dockerRegistryCredential2.URL {
						return nil, nil, errors.New("there is more than one credential for the registry: " + dockerRegistry.URL)
					}
				}
				// Check if there are valid credentials for the registry
				if dockerRegistryCredential.URL == dockerRegistry.URL {
					existCredentials = true
					err := validateStruct(dockerRegistryCredential)
					if err != nil {
						return nil, nil, errors.Wrap(err, "there aren't valid credentials for the registry: "+dockerRegistry.URL)
					}
					registryMap := structs.Map(dockerRegistryCredential)
					resultDockerRegistries = append(resultDockerRegistries, commons.ConvertMapKeysToSnakeCase(registryMap))
					if dockerRegistry.KeosRegistry {
						resultKeosRegistry = convertToMapStringString(registryMap)
					}
				}
			}

			if !existCredentials {
				return nil, nil, errors.New("there aren't valid credentials for the registry: " + dockerRegistry.URL)
			}
		}
		if dockerRegistry.KeosRegistry {
			// Check if there are more than one docker_registry defined as keos_registry
			keosCount++
			if keosCount > 1 {
				return nil, nil, errors.New("there are more than one docker_registry defined as keos_registry")
			}
		}
	}
	if keosCount == 0 {
		return nil, nil, errors.New("there isn't any docker_registry defined as keos_registry")
	}
	return resultKeosRegistry, resultDockerRegistries, nil
}

func validateGithubToken(secrets commons.Secrets, spec commons.Spec) (string, error) {
	var githubToken string
	var isGithubToken = regexp.MustCompile(`^(github_pat_|ghp_)\w+$`).MatchString

	if secrets.GithubToken != "" {
		githubToken = secrets.GithubToken
	} else if spec.Credentials.GithubToken != "" {
		githubToken = spec.Credentials.GithubToken
	} else {
		return "", nil
	}

	if isGithubToken(githubToken) {
		return githubToken, nil
	} else {
		return "", errors.New("github_token is not valid")
	}
}
