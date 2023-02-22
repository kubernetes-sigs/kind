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

package createworker

import (
	"bytes"
	"unicode"

	"io/ioutil"
	"os"
	"reflect"
	"strings"

	"github.com/fatih/structs"
	"github.com/oleiade/reflections"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions/cluster"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"

	vault "github.com/sosedoff/ansible-vault-go"
)

const secretName = "secrets.yml"
const secretPath = "./" + secretName

func encryptFile(filePath string, vaultPassword string) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}
	err = vault.EncryptFile(filePath, string(data), vaultPassword)
	if err != nil {
		return err
	}
	return nil
}

func decryptFile(filePath string, vaultPassword string) (string, error) {
	data, err := vault.DecryptFile(filePath, vaultPassword)
	if err != nil {
		return "", err
	}
	return data, nil
}

func convertToMapStringString(m map[string]interface{}) map[string]string {
	var m2 = map[string]string{}
	for k, v := range m {
		m2[k] = v.(string)
	}
	return m2
}

func convertStringMapToInterfaceMap(inputMap map[string]string) map[string]interface{} {
	outputMap := make(map[string]interface{})
	for key, value := range inputMap {
		outputMap[key] = value
	}
	return outputMap
}

func getSecrets(descriptorFile cluster.DescriptorFile, vaultPassword string) (map[string]string, map[string]string, string, []map[string]interface{}, error) {

	var c = map[string]string{}
	var r = map[string]string{}
	var dr = []map[string]interface{}{}
	var resultCreds = map[string]string{}
	var resultExternalReg = map[string]string{}
	var resultGHT string
	var infraProvider = descriptorFile.InfraProvider
	var resultDockerRegistries = []map[string]interface{}{}

	_, err := os.Stat("./secrets.yml")
	if err != nil {
		dc, err := reflections.GetField(descriptorFile.Credentials, strings.ToUpper(infraProvider))
		if err != nil {
			return c, r, "", dr, err
		}
		if reflect.DeepEqual(dc, reflect.Zero(reflect.TypeOf(dc)).Interface()) {
			return c, r, "", dr, errors.New("No " + infraProvider + " credentials found in secrets file and descriptor file")
		}
		if descriptorFile.Credentials.GithubToken == "" {
			return c, r, "", dr, errors.New("No GithubToken credentials found in secrets file and descriptor file")
		}
		for _, reg := range descriptorFile.DockerRegistries {
			for _, regCreds := range descriptorFile.Credentials.DockerRegistries {
				if reg.URL == regCreds.URL && regCreds.User != "" && regCreds.Pass != "" && reg.AuthRequired {
					dockerReg := structs.Map(regCreds)
					resultDockerRegistries = append(resultDockerRegistries, convertMapKeysToSnakeCase(dockerReg))
					if reg.KeosRegistry {
						r = map[string]string{"User": regCreds.User, "Pass": regCreds.Pass, "Url": regCreds.URL}
					}
				}
			}
		}

		m := structs.Map(dc)
		resultCreds = convertToMapStringString(m)
		resultExternalReg = r
		resultGHT = descriptorFile.Credentials.GithubToken

	} else {
		var secretFile SecretsFile
		secretRaw, err := decryptFile("./secrets.yml", vaultPassword)
		if err != nil {
			return c, r, "", dr, errors.New("The Vault password is incorrect")
		}
		err = yaml.Unmarshal([]byte(secretRaw), &secretFile)
		if err != nil {
			return c, r, "", dr, err
		}
		f, err := reflections.GetField(secretFile.Secrets, strings.ToUpper(descriptorFile.InfraProvider))
		if err != nil {
			return c, r, "", dr, err
		}
		if reflect.DeepEqual(f, reflect.Zero(reflect.TypeOf(f)).Interface()) {
			dc, err := reflections.GetField(descriptorFile.Credentials, strings.ToUpper(infraProvider))
			if err != nil {
				return c, r, "", dr, err
			}
			if reflect.DeepEqual(dc, reflect.Zero(reflect.TypeOf(dc)).Interface()) {
				return c, r, "", dr, errors.New("No " + infraProvider + " credentials found in secrets file and descriptor file")
			}
			resultCredsMap := structs.Map(dc)
			resultCreds = convertToMapStringString(resultCredsMap)

		} else {
			m := structs.Map(f)
			resultCreds = convertToMapStringString(m["Credentials"].(map[string]interface{}))
		}
		if secretFile.Secrets.GithubToken == "" {
			if descriptorFile.Credentials.GithubToken == "" {
				return c, r, "", dr, errors.New("No Github Token found in secrets file and descriptor file")
			}

			resultGHT = descriptorFile.Credentials.GithubToken
		} else {
			resultGHT = secretFile.Secrets.GithubToken
		}
		if secretFile.Secrets.ExternalRegistryCredentials == (cluster.ExternalRegistryCredentials{}) {
			if descriptorFile.Credentials.ExternalRegistry != (cluster.ExternalRegistryCredentials{}) {
				resultRegMap := structs.Map(descriptorFile.Credentials.ExternalRegistry)
				resultExternalReg = convertToMapStringString(resultRegMap)

			}
		} else {
			resultRegMap := structs.Map(secretFile.Secrets.ExternalRegistryCredentials)
			resultExternalReg = convertToMapStringString(resultRegMap)
		}

		if len(secretFile.Secrets.DockerRegistries) == 0 {
			if len(descriptorFile.DockerRegistries) > 0 {
				for _, registry := range descriptorFile.DockerRegistries {
					dockerReg := structs.Map(registry)
					resultDockerRegistries = append(resultDockerRegistries, convertMapKeysToSnakeCase(dockerReg))
				}
			}
		} else {
			for _, registry := range secretFile.Secrets.DockerRegistries {
				dockerReg := structs.Map(registry)
				resultDockerRegistries = append(resultDockerRegistries, convertMapKeysToSnakeCase(dockerReg))
			}
		}
	}
	return resultCreds, resultExternalReg, resultGHT, resultDockerRegistries, nil
}

func ensureSecretsFile(descriptorFile cluster.DescriptorFile, vaultPassword string) error {
	edited := false
	credentials, externalRegistry, github_token, dockerRegistries, err := getSecrets(descriptorFile, vaultPassword)
	if err != nil {
		return err
	}

	_, err = os.Stat(secretPath)
	if err != nil {
		secretMap := map[string]interface{}{}
		if github_token != "" {
			secretMap["github_token"] = github_token
		}
		if len(credentials) > 0 {
			creds := convertStringMapToInterfaceMap(credentials)
			creds = convertMapKeysToSnakeCase(creds)
			secretMap[descriptorFile.InfraProvider] = map[string]interface{}{
				"credentials": creds,
			}
		}

		if len(externalRegistry) > 0 {
			externalReg := convertStringMapToInterfaceMap(externalRegistry)
			externalReg = convertMapKeysToSnakeCase(externalReg)
			secretMap["external_registry"] = externalReg
		}

		if len(dockerRegistries) > 0 {
			for i, dockerReg := range dockerRegistries {
				dockerRegistries[i] = convertMapKeysToSnakeCase(dockerReg)
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

	if secretMap["secrets"][descriptorFile.InfraProvider] == nil && len(credentials) > 0 {
		edited = true
		creds := convertStringMapToInterfaceMap(credentials)
		creds = convertMapKeysToSnakeCase(creds)
		secretMap["secrets"][descriptorFile.InfraProvider] = map[string]interface{}{
			"credentials": creds}
	}

	if secretMap["secrets"]["external_registry"] == nil && len(externalRegistry) > 0 {
		edited = true
		externalReg := convertStringMapToInterfaceMap(externalRegistry)
		externalReg = convertMapKeysToSnakeCase(externalReg)
		secretMap["secrets"]["external_registry"] = externalReg
	}
	if secretMap["secrets"]["github_token"] == nil && github_token != "" {
		edited = true
		secretMap["secrets"]["github_token"] = github_token
	}
	if secretMap["secrets"]["docker_registries"] == nil && len(dockerRegistries) > 0 {
		edited = true
		for i, dockerReg := range dockerRegistries {
			dockerRegistries[i] = convertMapKeysToSnakeCase(dockerReg)
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

func convertMapToStruct(m map[string]interface{}, s interface{}) error {
	stValue := reflect.ValueOf(s).Elem()
	sType := stValue.Type()
	for i := 0; i < sType.NumField(); i++ {
		field := sType.Field(i)
		if value, ok := m[field.Name]; ok {
			stValue.Field(i).Set(reflect.ValueOf(value))
		}
	}
	return nil
}

func getMap(s interface{}) (map[string]string, error) {
	var resultMap map[string]string
	resultBytes, err := yaml.Marshal(s)
	if err != nil {
		return nil, err
	}
	yaml.Unmarshal(resultBytes, &resultMap)
	return resultMap, nil

}

func cleanStruct(m map[string]string) map[string]string {
	for k, v := range m {
		if v == "" {
			delete(m, k)
		}
	}
	return m
}

func rewriteDescriptorFile(descriptorName string) error {

	descriptorRAW, err := os.ReadFile("./" + descriptorName)

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

	err = ioutil.WriteFile(descriptorName, b, 0644)
	if err != nil {
		return err
	}

	return nil

}

func integrateClusterAutoscaler(node nodes.Node, kubeconfigPath string, clusterID string, provider string) exec.Cmd {
	cmd := node.Command("helm", "install", "cluster-autoscaler", "autoscaler/cluster-autoscaler",
		"--kubeconfig", kubeconfigPath,
		"--namespace", "kube-system",
		"--set", "autoDiscovery.clusterName="+clusterID,
		"--set", "autoDiscovery.labels[0].namespace=cluster-"+clusterID,
		"--set", "cloudProvider="+provider,
		"--set", "clusterAPIMode=incluster-incluster")

	return cmd
}

func getDecryptedSecret(vaultPassword string) (SecretsFile, error) {
	secretRaw, err := decryptFile("./secrets.yml", vaultPassword)
	secretFile := new(SecretsFile)
	if err != nil {
		err := errors.New("The vaultPassword is incorrect")
		return *secretFile, err
	} else {
		err = yaml.Unmarshal([]byte(secretRaw), &secretFile)
		if err != nil {
			return *secretFile, err
		}
		return *secretFile, nil
	}
}

func encryptSecret(secretMap map[string]map[string]interface{}, vaultPassword string) error {

	var b bytes.Buffer
	yamlEncoder := yaml.NewEncoder(&b)
	yamlEncoder.SetIndent(2)
	yamlEncoder.Encode(&secretMap)

	err := vault.EncryptFile(secretPath, string(b.Bytes()), vaultPassword)
	if err != nil {
		return err
	}

	return nil
}

func removeKey(nodes []*yaml.Node, key string) []*yaml.Node {
	newNodes := []*yaml.Node{}
	for i, node := range nodes {
		if node.Kind == yaml.MappingNode {
			j := 0
			for j < len(node.Content)/2 {
				if node.Content[j*2].Value == key {
					if i == 0 {
						// This is a root key, so remove it and its value.
						node.Content = append(node.Content[:j*2], node.Content[j*2+2:]...)
						continue
					}
					// This is not a root key, so keep it.
					j += 2
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

// getNode returns the first control plane
func getNode(ctx *actions.ActionContext) (nodes.Node, error) {
	allNodes, err := ctx.Nodes()
	if err != nil {
		return nil, err
	}

	controlPlanes, err := nodeutils.ControlPlaneNodes(allNodes)
	if err != nil {
		return nil, err
	}
	return controlPlanes[0], nil
}

func executeCommand(node nodes.Node, command string, envVars ...[]string) error {
	raw := bytes.Buffer{}
	cmd := node.Command("sh", "-c", command)
	if len(envVars) > 0 {
		cmd.SetEnv(envVars[0]...)
	}
	if err := cmd.SetStdout(&raw).Run(); err != nil {
		return err
	}
	return nil
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

func convertMapKeysToSnakeCase(m map[string]interface{}) map[string]interface{} {

	newMap := make(map[string]interface{})
	for k, v := range m {
		newKey := snakeCase(k)
		newMap[newKey] = v
	}
	return newMap
}
