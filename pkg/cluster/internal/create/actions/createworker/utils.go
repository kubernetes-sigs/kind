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
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	b64 "encoding/base64"

	"gopkg.in/yaml.v3"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions/cluster"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/exec"

	vault "github.com/sosedoff/ansible-vault-go"
)

const secretName = "secrets.yml"
const secretPath = "./" + secretName

func encryptFile(filePath string, vaultPassword string) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	err = vault.EncryptFile(filePath, string(data), vaultPassword)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return nil
}

func decryptFile(filePath string, vaultPassword string) (string, error) {
	data, err := vault.DecryptFile(filePath, vaultPassword)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	return data, nil
}

func generateB64Credentials(access_key string, secret_key string, region string) string {
	credentialsINIlines := "[default]\naws_access_key_id = " + access_key + "\naws_secret_access_key = " + secret_key + "\nregion = " + region + "\n\n"
	return b64.StdEncoding.EncodeToString([]byte(credentialsINIlines))
}

func getCredentials(descriptorFile cluster.DescriptorFile, vaultPassword string) (cluster.AWSCredentials, string, error) {
	awsEmptyCreds := cluster.AWSCredentials{}
	descriptorEmptyCreds := checkCreds(descriptorFile.AWSCredentials, descriptorFile.GithubToken)
	_, err := os.Stat(secretPath)
	if err != nil {

		if !descriptorEmptyCreds {
			return descriptorFile.AWSCredentials, descriptorFile.GithubToken, nil
		}
		err := errors.New("Incorrect AWS credentials or GithubToken in descriptor file")
		return awsEmptyCreds, "", err

	} else {

		secretFile, err := getDecryptedSecret(vaultPassword)
		secretsEmptyCreds := checkCreds(secretFile.Secret.AWSCredentials, secretFile.Secret.GithubToken)
		if secretsEmptyCreds {
			if !descriptorEmptyCreds {
				return descriptorFile.AWSCredentials, descriptorFile.GithubToken, nil
			}
			return awsEmptyCreds, "", errors.New("It is not possible to find the AWSCredentials in the descriptor or in secrets.yml")
		}
		return secretFile.Secret.AWSCredentials, secretFile.Secret.GithubToken, err

	}

}

func checkCreds(awsCreds cluster.AWSCredentials, github_token string) bool {
	awsEmptyCreds := cluster.AWSCredentials{}
	return awsCreds == awsEmptyCreds || github_token == ""
}

func ensureSecretsFile(descriptorFile cluster.DescriptorFile, vaultPassword string) error {
	edited := false

	awsCredentials, github_token, err := getCredentials(descriptorFile, vaultPassword)
	if err != nil {
		return err
	}

	_, err = os.Stat(secretPath)
	if err != nil {
		secret := Secret{awsCredentials, github_token}
		secretFile := SecretsFile{secret}
		secretRaw, err := yaml.Marshal(secretFile)
		secretMap := map[string]map[string]interface{}{}
		err = yaml.Unmarshal([]byte(secretRaw), &secretMap)
		if err != nil {
			return err
		}

		err = encryptSecret(secretMap, vaultPassword)
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

func rewriteDescriptorFile(descriptorName string) error {

	descriptorRAW, err := os.ReadFile("./" + descriptorName)
	if err != nil {
		return err
	}

	descriptorMap := map[string]interface{}{}

	err = yaml.Unmarshal(descriptorRAW, &descriptorMap)
	if err != nil {
		return err
	}

	if descriptorMap["aws"] != nil || descriptorMap["github_token"] != nil {
		deleteKey("aws", descriptorMap)
		deleteKey("github_token", descriptorMap)

		d, err := yaml.Marshal(&descriptorMap)
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(descriptorName, d, 0755)
		if err != nil {
			return err
		}

	}

	return nil

}

func deleteKey(key string, descriptorMap map[string]interface{}) {
	value := descriptorMap[key]
	if value != nil {
		delete(descriptorMap, key)
	}
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
			fmt.Println(err)
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
