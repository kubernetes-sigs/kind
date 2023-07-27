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

package commons

import (
	"bytes"
	"context"
	"unicode"

	"os"
	"strings"

	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"

	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	vault "github.com/sosedoff/ansible-vault-go"
)

const secretName = "secrets.yml"
const secretPath = "./" + secretName

func decryptFile(filePath string, vaultPassword string) (string, error) {
	data, err := vault.DecryptFile(filePath, vaultPassword)
	if err != nil {
		return "", err
	}
	var secret SecretsFile
	_ = yaml.Unmarshal([]byte(data), &secret)
	return data, nil
}

func convertStringMapToInterfaceMap(inputMap map[string]string) map[string]interface{} {
	outputMap := make(map[string]interface{})
	for key, value := range inputMap {
		outputMap[key] = value
	}
	return outputMap
}

func EnsureSecretsFile(spec Spec, vaultPassword string, clusterCredentials ClusterCredentials) error {
	var err error

	edited := false

	credentials := clusterCredentials.ProviderCredentials
	externalRegistry := clusterCredentials.KeosRegistryCredentials
	dockerRegistries := clusterCredentials.DockerRegistriesCredentials
	github_token := clusterCredentials.GithubToken

	_, err = os.Stat(secretPath)
	if err != nil {
		secretMap := map[string]interface{}{}
		if github_token != "" {
			secretMap["github_token"] = github_token
		}
		if len(credentials) > 0 {
			creds := convertStringMapToInterfaceMap(credentials)
			creds = ConvertMapKeysToSnakeCase(creds)
			secretMap[spec.InfraProvider] = map[string]interface{}{"credentials": creds}
		}

		if len(externalRegistry) > 0 {
			externalReg := convertStringMapToInterfaceMap(externalRegistry)
			externalReg = ConvertMapKeysToSnakeCase(externalReg)
			secretMap["external_registry"] = externalReg
		}

		if len(dockerRegistries) > 0 {
			for i, dockerReg := range dockerRegistries {
				dockerRegistries[i] = ConvertMapKeysToSnakeCase(dockerReg)
			}
			secretMap["docker_registries"] = dockerRegistries
		}

		secretFileMap := map[string]map[string]interface{}{
			"secrets": secretMap,
		}

		err = encryptSecret(secretFileMap, vaultPassword)
		if err != nil {
			return err
		}
		return nil
	}
	// En caso de que exista
	secretRaw, err := decryptFile(secretPath, vaultPassword)
	if err != nil {
		return err
	}
	secretMap := map[string]map[string]interface{}{}
	err = yaml.Unmarshal([]byte(secretRaw), &secretMap)
	if err != nil {
		return err
	}

	if secretMap["secrets"][spec.InfraProvider] == nil && len(credentials) > 0 {
		edited = true
		creds := convertStringMapToInterfaceMap(credentials)
		creds = ConvertMapKeysToSnakeCase(creds)
		secretMap["secrets"][spec.InfraProvider] = map[string]interface{}{"credentials": creds}
	}

	if secretMap["secrets"]["external_registry"] == nil && len(externalRegistry) > 0 {
		edited = true
		externalReg := convertStringMapToInterfaceMap(externalRegistry)
		externalReg = ConvertMapKeysToSnakeCase(externalReg)
		secretMap["secrets"]["external_registry"] = externalReg
	}
	if secretMap["secrets"]["github_token"] == nil && github_token != "" {
		edited = true
		secretMap["secrets"]["github_token"] = github_token
	}
	if secretMap["secrets"]["docker_registries"] == nil && len(dockerRegistries) > 0 {
		edited = true
		for i, dockerReg := range dockerRegistries {
			dockerRegistries[i] = ConvertMapKeysToSnakeCase(dockerReg)
		}
		secretMap["secrets"]["docker_registries"] = dockerRegistries
	}
	if edited {
		err = encryptSecret(secretMap, vaultPassword)
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

func RewriteDescriptorFile(descriptorPath string) error {

	descriptorRAW, err := os.ReadFile(descriptorPath)

	if err != nil {
		return err
	}

	var data yaml.Node
	err = yaml.Unmarshal(descriptorRAW, &data)
	if err != nil {
		return err
	}

	yamlNodes := removeKey(data.Content, "credentials")

	b, err := yaml.Marshal(yamlNodes[0])
	if err != nil {
		return err
	}

	err = os.WriteFile(descriptorPath, []byte(b), 0644)
	if err != nil {
		return err
	}

	return nil

}

func encryptSecret(secretMap map[string]map[string]interface{}, vaultPassword string) error {

	var b bytes.Buffer
	yamlEncoder := yaml.NewEncoder(&b)
	yamlEncoder.SetIndent(2)
	yamlEncoder.Encode(&secretMap)

	err := vault.EncryptFile(secretPath, b.String(), vaultPassword)
	if err != nil {
		return err
	}

	return nil
}

func removeKey(nodes []*yaml.Node, key string) []*yaml.Node {
	newNodes := []*yaml.Node{}
	for _, node := range nodes {
		if node.Kind == yaml.MappingNode {
			j := 0
			for j < len(node.Content)/2 {
				if node.Content[j*2].Value == key {
					node.Content = append(node.Content[:j*2], node.Content[j*2+2:]...)
					continue
				}
				j++
			}
			node.Content = removeKey(node.Content, key)
		}
		if node.Kind == yaml.SequenceNode {
			node.Content = removeKey(node.Content, key)
		}
		newNodes = append(newNodes, node)
	}
	return newNodes
}

func ExecuteCommand(n nodes.Node, command string, envVars ...[]string) (string, error) {
	raw := bytes.Buffer{}
	cmd := n.Command("sh", "-c", command)
	if len(envVars) > 0 {
		cmd.SetEnv(envVars[0]...)
	}
	if err := cmd.SetStdout(&raw).SetStderr(&raw).Run(); err != nil {
		return "", err
	}
	if strings.Contains(raw.String(), "Error:") {
		return "", errors.New(raw.String())
	}
	return raw.String(), nil
}

func snakeCase(s string) string {
	var result []rune
	for i, c := range s {
		if unicode.IsUpper(c) {
			if i > 0 && !unicode.IsUpper(rune(s[i-1])) {
				result = append(result, '_')
			}
			result = append(result, unicode.ToLower(c))
		} else {
			result = append(result, c)
		}
	}
	return string(result)
}

func ConvertMapKeysToSnakeCase(m map[string]interface{}) map[string]interface{} {
	newMap := make(map[string]interface{})
	for k, v := range m {
		newKey := snakeCase(k)
		newMap[newKey] = v
	}
	return newMap
}

// contains checks if a string is present in a slice
func Contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func AWSGetConfig(secrets map[string]string, region string) (aws.Config, error) {
	customProvider := credentials.NewStaticCredentialsProvider(
		secrets["AccessKey"], secrets["SecretKey"], "",
	)
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithCredentialsProvider(customProvider),
		config.WithRegion(region),
	)
	if err != nil {
		return aws.Config{}, err
	}
	return cfg, nil
}

func AWSIsPrivateSubnet(ctx context.Context, svc *ec2.Client, subnetID *string) (bool, error) {
	keyname := "association.subnet-id"
	drtInput := &ec2.DescribeRouteTablesInput{
		Filters: []types.Filter{
			{
				Name:   &keyname,
				Values: []string{*subnetID},
			},
		},
	}
	rt, err := svc.DescribeRouteTables(ctx, drtInput)
	if err != nil {
		return false, err
	}

	for _, associatedRouteTable := range rt.RouteTables {
		for i := range associatedRouteTable.Routes {
			route := associatedRouteTable.Routes[i]
			// Check if route is public
			if route.DestinationCidrBlock != nil &&
				route.GatewayId != nil &&
				*route.DestinationCidrBlock == "0.0.0.0/0" &&
				strings.Contains(*route.GatewayId, "igw") {
				return false, nil // Public subnet
			}
		}
	}

	return true, nil
}

func AWSGetPrivateAZs(ctx context.Context, svc *ec2.Client, subnets []Subnets) ([]string, error) {
	var azs []string
	for _, s := range subnets {
		isPrivate, err := AWSIsPrivateSubnet(ctx, svc, &s.SubnetId)
		if err != nil {
			return nil, nil
		}
		if isPrivate {
			sid := &ec2.DescribeSubnetsInput{
				SubnetIds: []string{s.SubnetId},
			}
			ds, err := svc.DescribeSubnets(ctx, sid)
			if err != nil {
				return nil, nil
			}
			for _, describeSubnet := range ds.Subnets {
				if !slices.Contains(azs, *describeSubnet.AvailabilityZone) {
					azs = append(azs, *describeSubnet.AvailabilityZone)
				}
			}
		}
	}
	return azs, nil
}

func AWSGetAZs(ctx context.Context, svc *ec2.Client) ([]string, error) {
	var azs []string
	result, err := svc.DescribeAvailabilityZones(ctx, &ec2.DescribeAvailabilityZonesInput{})
	if err != nil {
		return nil, err
	}
	for i, az := range result.AvailabilityZones {
		if i == 3 {
			break
		}
		azs = append(azs, *az.ZoneName)
	}
	return azs, nil
}
