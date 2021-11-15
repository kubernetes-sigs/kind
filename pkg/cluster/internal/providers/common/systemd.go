/*
Copyright 2021 The Kubernetes Authors.

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

package common

import (
	"bufio"
	"context"
	"os"
	"regexp"
	"sync"

	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

var systemdReachedMultiUserSystemRegexp *regexp.Regexp
var systemdReachedMultiUserSystemRegexpCompileOnce sync.Once

func SystemdReachedMultiUserSystemRegexp() *regexp.Regexp {
	systemdReachedMultiUserSystemRegexpCompileOnce.Do(func() {
		systemdReachedMultiUserSystemRegexp = regexp.MustCompile("Reached target .*Multi-User System.*|detected cgroup v1")
	})
	return systemdReachedMultiUserSystemRegexp
}

func WaitUntilLogRegexpMatches(logCtx context.Context, logCmd exec.Cmd, re *regexp.Regexp) error {
	pr, pw, err := os.Pipe()
	if err != nil {
		return err
	}
	logCmd.SetStdout(pw)
	logCmd.SetStderr(pw)

	defer pr.Close()
	cmdErrC := make(chan error, 1)
	go func() {
		defer pw.Close()
		cmdErrC <- logCmd.Run()
	}()

	sc := bufio.NewScanner(pr)
	for sc.Scan() {
		line := sc.Text()
		if re.MatchString(line) {
			return nil
		}
	}

	// when we timeout the process will have been killed due to the timeout, which is not interesting
	// in other cases if the command errored this may be a useful error
	if ctxErr := logCtx.Err(); ctxErr != context.DeadlineExceeded {
		if cmdErr := <-cmdErrC; cmdErr != nil {
			return errors.Wrap(cmdErr, "failed to read logs")
		}
	}
	// otherwise generic error
	return errors.Errorf("could not find a log line that matches %q", re.String())
}
