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

package common

import (
	"bufio"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// /proc/meminfo is used by to report the amount of free and used memory
// on the system as well as the shared memory and buffers used by the kernel.
const memFile = "/proc/meminfo"

// GetSystemMemTotal returns the total number of memory available on the system in bytes
// It returns 0 if it can not obtain the memory
func GetSystemMemTotal() uint64 {
	file, err := os.Open(memFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ":")
		if len(fields) != 2 {
			continue
		}
		key := strings.TrimSpace(fields[0])
		if key == "MemTotal" {
			value := strings.TrimSpace(fields[1])
			value = strings.Replace(value, " kB", "", -1)
			t, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return 0
			}
			return t * 1024
		}
	}
	return 0
}

// GetSystemCPUs return the number of CPUs in the system
func GetSystemCPUs() int {
	return runtime.NumCPU()
}
