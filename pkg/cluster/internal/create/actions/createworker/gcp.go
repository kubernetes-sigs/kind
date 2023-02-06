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

// Package createworker implements the create worker action
package createworker

import (
	b64 "encoding/base64"
	"encoding/json"
	"net/url"
)

func getGCPEnv(credentials map[string]string, githubToken string) []string {
	data := map[string]interface{}{
		"type":                        "service_account",
		"project_id":                  credentials["ProjectID"],
		"private_key_id":              credentials["PrivateKeyID"],
		"private_key":                 credentials["PrivateKey"],
		"client_email":                credentials["ClientEmail"],
		"client_id":                   credentials["ClientID"],
		"auth_uri":                    "https://accounts.google.com/o/oauth2/auth",
		"token_uri":                   "https://accounts.google.com/o/oauth2/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url":        "https://www.googleapis.com/robot/v1/metadata/x509/" + url.QueryEscape(credentials["ClientEmail"]),
	}

	jsonData, _ := json.Marshal(data)
	envVars := []string{
		"GCP_B64ENCODED_CREDENTIALS=" + b64.StdEncoding.EncodeToString([]byte(jsonData)),
		"GITHUB_TOKEN=" + githubToken,
	}
	return envVars
}
