/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or impliep.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package common contains common code for implementing providers
package common

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/disk"
	"sigs.k8s.io/kind/pkg/errors"
	"sigs.k8s.io/kind/pkg/exec"
)

func CheckFreeDiskSpace(maxPercentage int, command string, args ...string) error {
	var buff bytes.Buffer
	if err := exec.Command(command, args...).SetStdout(&buff).Run(); err != nil {
		return err
	}

	path := buff.String()
	path = strings.ReplaceAll(path, "\n", "")
	path = strings.ReplaceAll(path, "\r\n", "")

	usageStat, err := disk.Usage(path)
	if err != nil {
		return err
	}

	usedPercent, err := strconv.Atoi(fmt.Sprintf("%2.f", usageStat.UsedPercent))
	if err != nil {
		return err
	}

	if usedPercent >= maxPercentage {
		return errors.Errorf("out of disk space: more than %d%% used", maxPercentage)
	}

	return nil
}
