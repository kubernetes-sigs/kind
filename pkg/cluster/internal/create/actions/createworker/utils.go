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
	fmt.Println("Decrypted: ")
	fmt.Println(data)
	return data, nil
}

func generateB64Credentials(access_key string, secret_key string, region string) string {
	credentialsINIlines := "[default]\naws_access_key_id = " + access_key + "\naws_secret_access_key = " + secret_key + "\nregion = " + region + "\n\n"
	return b64.StdEncoding.EncodeToString([]byte(credentialsINIlines))
}

func getCredentials(descriptorFile DescriptorFile, vaultPassword string) (AWS, error) {
	aws := AWS{}

	_, err := os.Stat("./secrets.yaml")
	if err != nil {
		fmt.Println("descriptorFile.AWS: ", descriptorFile.AWS)
		if aws != descriptorFile.AWS {
			rewriteDescriptorFile(descriptorFile)
			return descriptorFile.AWS, nil
		}
		err := errors.New("Incorrect AWS credentials in Cluster.yaml")
		return aws, err

	} else {
		secretRaw, err := decryptFile("./secrets.yaml", vaultPassword)
		var secretFile SecretFile
		if err != nil {
			err := errors.New("The vaultPassword is incorrect")
			return aws, err
		} else {
			fmt.Println("secretRAW: ")
			fmt.Println(secretRaw)
			err = yaml.Unmarshal([]byte(secretRaw), &secretFile)
			if err != nil {
				fmt.Println(err)
				return aws, err
			}
			fmt.Println("secretFile: ", secretFile)
			fmt.Println("secretFile.AWS: ", secretFile.Secrets.AWS)
			return secretFile.Secrets.AWS, nil
		}
	}

}

func stringToBytes(str string) []byte {
	buf := &bytes.Buffer{}
	gob.NewEncoder(buf).Encode(str)
	bytes := buf.Bytes()

	return bytes
}

func rewriteDescriptorFile(descriptorFile DescriptorFile) error {
	descriptor := DescriptorFile{}
	viper.SetConfigName("cluster.yaml")
	currentDir, err := currentdir()
	if err != nil {
		fmt.Println(err)
		return err
	}
	viper.AddConfigPath(currentDir)
	err = viper.Unmarshal(&descriptor)
	if err != nil {
		fmt.Println("unable to decode into struct, %v", err)
	}

	fmt.Println(descriptor)
	descriptor = descriptorFile
	descriptor.AWS = AWS{}
	d, err := yaml.Marshal(&descriptor)
	if err != nil {
		fmt.Println("error: %v", err)
		return err
	}

	fmt.Println(string(d))

	// write to file
	f, err := os.Create(currentDir + "/cluster.yaml")
	if err != nil {
		fmt.Println(err)
		return nil
	}

	err = ioutil.WriteFile("cluster.yaml", d, 0755)
	if err != nil {
		fmt.Println("error: %v", err)
		return err
	}

	f.Close()

	return nil

}
