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

import "testing"

func TestPortOrGetFreePort(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		port    int32
		want    int32
		wantErr bool
	}{
		{
			name:    "Valid port",
			port:    80,
			want:    80,
			wantErr: false,
		},
		{
			name:    "No port",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := PortOrGetFreePort(tt.port, "localhost")
			if (err != nil) != tt.wantErr {
				t.Errorf("PortOrGetFreePort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want != 0 && got != tt.want {
				t.Errorf("PortOrGetFreePort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetFreePort(t *testing.T) {
	tests := []struct {
		name       string
		listenAddr string
		wantErr    bool
	}{
		{
			name:       "listen on localhost",
			listenAddr: "localhost",
			wantErr:    false,
		},
		{
			name:       "listen on IPv4 localhost address",
			listenAddr: "127.0.0.1",
			wantErr:    false,
		},
		{
			name:       "listen on IPv4 non existent address",
			listenAddr: "88.88.88.0",
			wantErr:    true,
		},
		{
			name:       "listen on IPv6 non existent address",
			listenAddr: "2112:beaf:beaf:2:3",
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := GetFreePort(tt.listenAddr)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFreePort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got < 0 || got > 65535 && err != nil {
				t.Errorf("GetFreePort() = %v is not a valid port number ", got)
			}
		})
	}
}
