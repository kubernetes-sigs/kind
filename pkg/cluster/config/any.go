/*
Copyright 2018 The Kubernetes Authors.

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

package config

/*
This file contains interfaces and code related to all API versions
*/

// Any represents any API version of Config
type Any interface {
	// Validate should return an error of type `*Errors` if config is invalid
	Validate() error
	// ToCurrent should convert a config version to the version in this package
	ToCurrent() *Config
	// ApplyDefaults should set unset fields to defaults
	ApplyDefaults()
	// Kind should return "Config"
	Kind() string
	// APIVersion should return the apiVersion for this config
	APIVersion() string
}
