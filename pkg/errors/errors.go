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

package errors

import (
	pkgerrors "github.com/pkg/errors"

	"sigs.k8s.io/kind/pkg/exec"
)

// Causer is an interface to github.com/pkg/errors error's Cause() wrapping
type Causer interface {
	// Cause returns the underlying error
	Cause() error
}

// StackTracer is an interface to github.com/pkg/errors error's StackTrace()
type StackTracer interface {
	// StackTrace returns the StackTrace ...
	StackTrace() pkgerrors.StackTrace
}

// RunError returns an exec.RunError if the error chain
// contains an exec.RunError
func RunError(err error) *exec.RunError {
	var runError *exec.RunError
	for {
		if rErr, ok := err.(*exec.RunError); ok {
			runError = rErr
		}
		if causerErr, ok := err.(Causer); ok {
			err = causerErr.Cause()
		} else {
			break
		}
	}
	return runError
}

// StackTrace returns the deepest StackTrace is a Cause chain
// https://github.com/pkg/errors/issues/173
func StackTrace(err error) pkgerrors.StackTrace {
	var stackErr error
	for {
		if _, ok := err.(StackTracer); ok {
			stackErr = err
		}
		if causerErr, ok := err.(Causer); ok {
			err = causerErr.Cause()
		} else {
			break
		}
	}
	if stackErr != nil {
		return stackErr.(StackTracer).StackTrace()
	}
	return nil
}
