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

package util

import "bytes"

// Errors implements error and contains all config errors
// This is returned by Config.Validate
type Errors []error

// NewErrors returns a new Errors from a slice of error
func NewErrors(errors []error) Errors {
	return errors
}

// assert Errors implements error
var _ error = Errors{}

// Error implements the error interface
func (e Errors) Error() string {
	var buff bytes.Buffer
	for _, err := range e {
		buff.WriteString(err.Error())
		buff.WriteRune('\n')
	}
	return buff.String()
}

// Errors returns the slice of errors contained by Errors
func (e Errors) Errors() []error {
	return e
}

// Flatten recursively flattens any Errors to a single top level Errors
func Flatten(errors Errors) Errors {
	flat := []error{}
	for _, err := range errors {
		if v, ok := err.(Errors); ok {
			flat = append(flat, Flatten(v))
		} else {
			flat = append(flat, err)
		}
	}
	return flat
}
