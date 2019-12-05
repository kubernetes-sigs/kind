/*
Copyright 2019 The Kubernetes Authors.

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

package common

import (
	"net"
)

// PortOrGetFreePort is a helper that either returns the provided port
// if valid or returns a new free port on listenAddr
func PortOrGetFreePort(port int32, listenAddr string) (int32, error) {
	if port > 0 {
		return port, nil
	}
	return GetFreePort(listenAddr)
}

// GetFreePort is a helper used to get a free TCP port on the host
func GetFreePort(listenAddr string) (int32, error) {
	dummyListener, err := net.Listen("tcp", net.JoinHostPort(listenAddr, "0"))
	if err != nil {
		return 0, err
	}
	defer dummyListener.Close()
	port := dummyListener.Addr().(*net.TCPAddr).Port
	return int32(port), nil
}
