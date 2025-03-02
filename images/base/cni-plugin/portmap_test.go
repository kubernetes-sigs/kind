/*
Copyright 2025 The Kubernetes Authors.

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

package main

import (
	"net/netip"
	"reflect"
	"testing"
)

func TestEncodeWithAlignment(t *testing.T) {
	testCases := []struct {
		name     string
		input    interface{}
		expected []byte
	}{
		{
			"Uint16 - Max",
			uint16(65535),
			[]byte{0xff, 0xff, 0x00, 0x00},
		},
		{
			"Uint16 - With padding",
			uint16(5678),
			[]byte{0x16, 0x2e, 0x00, 0x00},
		},
		{
			"Uint32 - Max",
			uint32(4294967295),
			[]byte{0xff, 0xff, 0xff, 0xff},
		},
		{
			"Uint32",
			uint32(5678),
			[]byte{0x00, 0x00, 0x16, 0x2e},
		},
		{
			"IPv4",
			netip.MustParseAddr("1.1.1.1").AsSlice(),
			[]byte{0x01, 0x01, 0x01, 0x01},
		},
		{
			"IPv6",
			netip.MustParseAddr("::").AsSlice(),
			[]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := encodeWithAlignment(tc.input)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Expected %x, but got %x", tc.expected, result)
			}
		})
	}
}
