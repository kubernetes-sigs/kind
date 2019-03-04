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

package util

import (
	"fmt"
	"runtime"
)

// GetOS validates/returns the current operating system if supported and panics otherwise
func GetOS() string {
	switch runtime.GOOS {
	case "darwin":
		return "darwin"
	case "linux":
		return "linux"
	case "windows":
		return "windows"
	}
	panic(fmt.Sprintf("unsupported OS %s", runtime.GOOS))
}

// GetArch validates/returns the current architecture if supported and panics otherwise
func GetArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	}
	panic(fmt.Sprintf("unsupported architecture %s", runtime.GOARCH))
}

// GetOSandArch validates/returns the current os/arch combination if supported and panics otherwise
func GetOSandArch(separator string) string {
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "amd64" {
			return "darwin" + separator + "amd64"
		}
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			return "linux" + separator + "amd64"
		case "arm64":
			return "linux" + separator + "arm64"
		}
	case "windows":
		if runtime.GOARCH == "amd64" {
			return "windows" + separator + "amd64"
		}
	}
	panic(fmt.Sprintf("unsupported platform %s%s%s", runtime.GOOS, separator, runtime.GOARCH))
}
