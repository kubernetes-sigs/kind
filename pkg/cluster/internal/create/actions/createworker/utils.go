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
	"strings"

	"github.com/fatih/structs"
	"github.com/oleiade/reflections"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions"
	"sigs.k8s.io/kind/pkg/cluster/internal/create/actions/cluster"
	"sigs.k8s.io/kind/pkg/cluster/nodes"
	"sigs.k8s.io/kind/pkg/cluster/nodeutils"
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
		return nil
	}
	err = vault.EncryptFile(filePath, string(data), vaultPassword)
	if err != nil {
		return nil
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

	_, err := os.Stat("./secrets.yml")
	if err != nil {
		if descriptorFile.Credentials == (cluster.Credentials{}) {
			return c, r, "", errors.New("Incorrect credentials in descriptor file")
		}
		m := structs.Map(descriptorFile.Credentials)
		r := map[string]string{"User": descriptorFile.ExternalRegistry.User, "Pass": descriptorFile.ExternalRegistry.Pass, "Url": descriptorFile.ExternalRegistry.URL}
		return convertToMapStringString(m), r, descriptorFile.GithubToken, nil
	} else {
		var secretFile SecretsFile
		secretRaw, err := decryptFile("./secrets.yml", vaultPassword)
		if err != nil {
			return c, r, "", errors.New("The Vault password is incorrect")
		} else {
			err = yaml.Unmarshal([]byte(secretRaw), &secretFile)
			if err != nil {
				return c, r, "", err
			}
			f, err := reflections.GetField(secretFile.Secrets, strings.ToUpper(descriptorFile.InfraProvider))
			if err != nil {
				return c, r, "", errors.New("No " + descriptorFile.InfraProvider + " credentials found in secrets file")
			}
			m := structs.Map(f)
			r := map[string]string{"User": secretFile.Secrets.ExternalRegistry.User, "Pass": secretFile.Secrets.ExternalRegistry.Pass, "Url": descriptorFile.ExternalRegistry.URL}
			return convertToMapStringString(m["Credentials"].(map[string]interface{})), r, secretFile.Secrets.GithubToken, nil
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

	if descriptorMap["credentials"] != nil || descriptorMap["github_token"] != nil {
		deleteKey("credentials", descriptorMap)
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
