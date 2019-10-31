/*
Copyright 2016 The Kubernetes Authors.

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

package kubeconfig

// NOTE this is from client-go. Rather than pull in client-go for this one
// standalone method, we have a fork here.
// https://github.com/kubernetes/client-go/blob/6d7018244d72350e2e8c4a19ccdbe4c8083a9143/util/homedir/homedir.go
// We've modified this to require injecting os.Getenv as a dependency for testing purposes

import (
	"os"
	"path/filepath"
	"runtime"
)

// homeDir returns the home directory for the current user.
// On Windows:
// 1. the first of %HOME%, %HOMEDRIVE%%HOMEPATH%, %USERPROFILE% containing a `.kube\config` file is returned.
// 2. if none of those locations contain a `.kube\config` file, the first of %HOME%, %USERPROFILE%, %HOMEDRIVE%%HOMEPATH% that exists and is writeable is returned.
// 3. if none of those locations are writeable, the first of %HOME%, %USERPROFILE%, %HOMEDRIVE%%HOMEPATH% that exists is returned.
// 4. if none of those locations exists, the first of %HOME%, %USERPROFILE%, %HOMEDRIVE%%HOMEPATH% that is set is returned.
func homeDir(getEnv func(string) string) string {
	if runtime.GOOS == "windows" {
		home := getEnv("HOME")
		homeDriveHomePath := ""
		if homeDrive, homePath := getEnv("HOMEDRIVE"), getEnv("HOMEPATH"); len(homeDrive) > 0 && len(homePath) > 0 {
			homeDriveHomePath = homeDrive + homePath
		}
		userProfile := getEnv("USERPROFILE")

		// Return first of %HOME%, %HOMEDRIVE%/%HOMEPATH%, %USERPROFILE% that contains a `.kube\config` file.
		// %HOMEDRIVE%/%HOMEPATH% is preferred over %USERPROFILE% for backwards-compatibility.
		for _, p := range []string{home, homeDriveHomePath, userProfile} {
			if len(p) == 0 {
				continue
			}
			if _, err := os.Stat(filepath.Join(p, ".kube", "config")); err != nil {
				continue
			}
			return p
		}

		firstSetPath := ""
		firstExistingPath := ""

		// Prefer %USERPROFILE% over %HOMEDRIVE%/%HOMEPATH% for compatibility with other auth-writing tools
		for _, p := range []string{home, userProfile, homeDriveHomePath} {
			if len(p) == 0 {
				continue
			}
			if len(firstSetPath) == 0 {
				// remember the first path that is set
				firstSetPath = p
			}
			info, err := os.Stat(p)
			if err != nil {
				continue
			}
			if len(firstExistingPath) == 0 {
				// remember the first path that exists
				firstExistingPath = p
			}
			if info.IsDir() && info.Mode().Perm()&(1<<(uint(7))) != 0 {
				// return first path that is writeable
				return p
			}
		}

		// If none are writeable, return first location that exists
		if len(firstExistingPath) > 0 {
			return firstExistingPath
		}

		// If none exist, return first location that is set
		if len(firstSetPath) > 0 {
			return firstSetPath
		}

		// We've got nothing
		return ""
	}
	return getEnv("HOME")
}
