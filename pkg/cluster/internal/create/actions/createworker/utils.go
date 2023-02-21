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

func getSecrets(descriptorFile cluster.DescriptorFile, vaultPassword string) (map[string]string, map[string]string, string, error) {

	var c = map[string]string{}
	var r = map[string]string{}
	var resultCreds = map[string]string{}
	var resultReg = map[string]string{}
	var resultGHT string

	_, err := os.Stat("./secrets.yml")
	if err != nil {
		if descriptorFile.Credentials == (cluster.Credentials{}) {
			return c, r, "", errors.Wrap(err, "Incorrect credentials in descriptor file")
		}
		m := structs.Map(descriptorFile.Credentials)
		r := map[string]string{"User": descriptorFile.ExternalRegistry.User, "Pass": descriptorFile.ExternalRegistry.Pass, "Url": descriptorFile.ExternalRegistry.URL}
		resultCreds = convertToMapStringString(m)
		resultReg = r
		resultGHT = descriptorFile.GithubToken
	} else {
		var secretFile SecretsFile
		secretRaw, err := decryptFile("./secrets.yml", vaultPassword)
		if err != nil {
			return c, r, "", errors.New("The Vault password is incorrect")
		}
		err = yaml.Unmarshal([]byte(secretRaw), &secretFile)
		if err != nil {
			return c, r, "", err
		}
		f, err := reflections.GetField(secretFile.Secrets, strings.ToUpper(descriptorFile.InfraProvider))
		if err != nil {
			return c, r, "", err
		}
		if reflect.DeepEqual(f, reflect.Zero(reflect.TypeOf(f)).Interface()) {
			if descriptorFile.Credentials == (cluster.Credentials{}) {
				return c, r, "", errors.New("No " + descriptorFile.InfraProvider + " credentials found in secrets file and descriptor file")
			}
			resultCredsMap := structs.Map(descriptorFile.Credentials)
			resultCreds = convertToMapStringString(resultCredsMap)

		} else {
			m := structs.Map(f)
			resultCreds = convertToMapStringString(m["Credentials"].(map[string]interface{}))
		}
		if secretFile.Secrets.GithubToken == "" {
			if descriptorFile.GithubToken == "" {
				return c, r, "", errors.New("No Github Token found in secrets file and descriptor file")
			}

			resultGHT = descriptorFile.GithubToken
		} else {
			resultGHT = secretFile.Secrets.GithubToken
		}
		if secretFile.Secrets.ExternalRegistry == (ExternalRegistry{}) {
			// TODO: Adaptar para que puedan ser multiples
			if descriptorFile.ExternalRegistry != (cluster.ExternalRegistry{}) {
				resultReg = map[string]string{"User": descriptorFile.ExternalRegistry.User, "Pass": descriptorFile.ExternalRegistry.Pass, "Url": descriptorFile.ExternalRegistry.URL}

			}
		} else {
			resultReg = map[string]string{"User": secretFile.Secrets.ExternalRegistry.User, "Pass": secretFile.Secrets.ExternalRegistry.Pass, "Url": descriptorFile.ExternalRegistry.URL}
		}
	}
	return resultCreds, resultReg, resultGHT, nil
}

func ensureSecretsFile(descriptorFile cluster.DescriptorFile, vaultPassword string) error {
	edited := false
	credentials, externalRegistry, github_token, err := getSecrets(descriptorFile, vaultPassword)
	if err != nil {
		return err
	}
	awsCredentials, gcpCredentials, secretExternalRegistry, err := fillCredentials(credentials, externalRegistry)
	if err != nil {
		return err
	}

	_, err = os.Stat(secretPath)
	if err != nil {
		secretMap := map[string]interface{}{}
		if github_token != "" {
			secretMap["github_token"] = github_token
		}
		if len(gcpCredentials) > 0 {
			secretMap["gcp"] = map[string]interface{}{
				"credentials": gcpCredentials,
			}
		}
		if len(awsCredentials) > 0 {
			secretMap["aws"] = map[string]interface{}{
				"credentials": awsCredentials,
			}
		}
		if secretExternalRegistry["user"] != "" && secretExternalRegistry["pass"] != "" {
			externalRegistryMap := map[string]interface{}{
				"user": secretExternalRegistry["user"],
				"pass": secretExternalRegistry["pass"],
				"url":  externalRegistry["url"],
			}
			secretMap["external_registry"] = externalRegistryMap
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

	if secretMap["secrets"]["aws"] == nil {
		edited = true
		secretMap["secrets"]["aws"] = awsCredentials
	}
	if secretMap["secrets"]["gcp"] == nil {
		edited = true
		secretMap["secrets"]["gcp"] = gcpCredentials
	}
	if secretMap["secrets"]["external_registry"] == nil {
		edited = true
		secretMap["secrets"]["external_registry"] = secretExternalRegistry
	}
	if secretMap["secrets"]["github_token"] == nil {
		edited = true
		secretMap["secrets"]["github_token"] = github_token
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

func fillCredentials(credentials map[string]string, externalRegistry map[string]string) (map[string]string, map[string]string, map[string]string, error) {
	awsCredentials := cluster.Credentials{}
	awsCredentials.AccessKey = credentials["AccessKey"]
	awsCredentials.SecretKey = credentials["SecretKey"]
	awsCredentials.Region = credentials["Region"]
	awsCredentials.Account = credentials["Account"]
	awsMap, err := getMap(awsCredentials)
	if err != nil {
		return nil, nil, nil, err
	}
	awsMap = cleanStruct(awsMap)

	gcpCredentials := cluster.Credentials{}
	gcpCredentials.ProjectID = credentials["ProjectID"]
	gcpCredentials.PrivateKeyID = credentials["PrivateKeyID"]
	gcpCredentials.PrivateKey = credentials["PrivateKey"]
	gcpCredentials.ClientEmail = credentials["ClientEmail"]
	gcpCredentials.ClientID = credentials["ClientID"]
	gcpMap, err := getMap(gcpCredentials)
	if err != nil {
		return nil, nil, nil, err
	}
	gcpMap = cleanStruct(gcpMap)

	secretExternalRegistry := ExternalRegistry{externalRegistry["User"], externalRegistry["Pass"]}
	secretExternalRegistryMap, err := getMap(secretExternalRegistry)
	if err != nil {
		return nil, nil, nil, err
	}

	return awsMap, gcpMap, secretExternalRegistryMap, nil
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
	yamlNodes = removeKey(yamlNodes, "github_token")

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
