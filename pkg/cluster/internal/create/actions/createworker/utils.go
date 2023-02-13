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
	gob "encoding/gob"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	b64 "encoding/base64"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions/cluster"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/exec"

	vault "github.com/sosedoff/ansible-vault-go"
)

func createDirectory(directory string) error {
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		err = os.Mkdir(directory, 0777)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}
	return nil
}

func currentdir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		return "", nil
	}

	return cwd, nil
}

func writeFile(filePath string, contentLines []string) error {
	f, err := os.Create(filePath)
	if err != nil {
		fmt.Println(err)
		f.Close()
		return nil
	}
	for _, v := range contentLines {
		fmt.Fprintf(f, v)
		if err != nil {
			fmt.Println(err)
			return nil
		}
	}
	err = f.Close()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return nil
}

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
	aws := cluster.AWSCredentials{}

	_, err := os.Stat("./secrets.yml")
	if err != nil {
		if aws != descriptorFile.AWSCredentials {
			return descriptorFile.AWSCredentials, descriptorFile.GithubToken, nil
		}
		err := errors.New("Incorrect AWS credentials in descriptor file")
		return aws, "", err

	} else {
		secretRaw, err := decryptFile("./secrets.yml", vaultPassword)
		var secretFile SecretsFile
		if err != nil {
			err := errors.New("The vaultPassword is incorrect")
			return aws, "", err
		} else {
			err = yaml.Unmarshal([]byte(secretRaw), &secretFile)
			if err != nil {
				fmt.Println(err)
				return aws, "", err
			}
			return secretFile.Secret.AWSCredentials, secretFile.Secret.GithubToken, nil
		}
	}

}

func stringToBytes(str string) []byte {
	buf := &bytes.Buffer{}
	gob.NewEncoder(buf).Encode(str)
	bytes := buf.Bytes()

	return bytes
}

func rewriteDescriptorFile(descriptorName string) error {

	descriptorRAW, err := os.ReadFile("./" + descriptorName)
	if err != nil {
		return err
	}

	descriptorMap := map[string]interface{}{}
	viper.SetConfigName(descriptorName)
	currentDir, err := currentdir()
	if err != nil {
		fmt.Println(err)
		return err
	}
	viper.AddConfigPath(currentDir)

	err = yaml.Unmarshal(descriptorRAW, &descriptorMap)
	if err != nil {
		return err
	}

	if descriptorMap["aws"] != nil || descriptorMap["github_token"] != nil {
		deleteKey("aws", descriptorMap)
		deleteKey("github_token", descriptorMap)

		d, err := yaml.Marshal(&descriptorMap)
		if err != nil {
			fmt.Println("error: %v", err)
			return err
		}

		// write to file
		f, err := os.Create(currentDir + descriptorName)
		if err != nil {
			fmt.Println(err)
			return nil
		}

		err = ioutil.WriteFile(descriptorName, d, 0755)
		if err != nil {
			fmt.Println("error: %v", err)
			return err
		}

		f.Close()

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
