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
	"fmt"
	"io/ioutil"
	"log"
	"net/netip"
	"os"
	"reflect"
	"testing"
)

func TestAllocator(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "temp")
	if err != nil {
		t.Errorf("create tempDir: %v", err)
	}
	t.Logf("logs on %s", tempDir)
	dbDir = tempDir
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	// initialize variables
	err = start()
	if err != nil {
		t.Fatal(err)
	}

	logger = log.New(os.Stdout, "", 0) // 0 flag for no timestamps/prefixes

	a, err := NewAllocator(netip.MustParsePrefix("192.168.1.0/25"))
	if err != nil {
		t.Fatal(err)
	}
	// can not allocate on the reserved space
	err = writeNetworkConfigWithoutIPs(&NetworkConfig{ContainerID: "id-static-1"})
	if err != nil {
		t.Fatal(err)
	}
	err = a.AllocateAddress(netip.MustParseAddr("192.168.1.2"), "id-static-1")
	if err == nil {
		t.Fatal("can not alllocate on the reserved space")
	}

	allocatable := a.size - uint64(a.reserved)
	var i uint64
	for i = 0; i < allocatable; i++ {
		id := fmt.Sprintf("id-%d", i)
		err = writeNetworkConfigWithoutIPs(&NetworkConfig{ContainerID: id})
		if err != nil {
			t.Fatal(err)
		}
		_, err := a.Allocate(id)
		if err != nil {
			t.Fatal(err)
		}
	}

	// it should fail to allocate since it is full
	err = writeNetworkConfigWithoutIPs(&NetworkConfig{ContainerID: "id-should-fail-1"})
	if err != nil {
		t.Fatal(err)
	}
	ip, err := a.Allocate("id-should-fail-1")
	if err == nil {
		t.Errorf("unexpected success %s", ip.String())
	}
	if a.Free() != 0 {
		t.Errorf("no free addresses expected, got %d", a.Free())
	}

	// release and allocate manually
	a.Release(netip.MustParseAddr("192.168.1.33"))
	if a.Free() != 1 {
		t.Errorf("one free addresses expected, got %d", a.Free())
	}
	err = writeNetworkConfigWithoutIPs(&NetworkConfig{ContainerID: "id-static-2"})
	if err != nil {
		t.Fatal(err)
	}
	err = a.AllocateAddress(netip.MustParseAddr("192.168.1.33"), "id-static-2")
	if err != nil {
		t.Error(err)
	}

	// it should fail to allocate since it is full
	err = writeNetworkConfigWithoutIPs(&NetworkConfig{ContainerID: "id-should-fail-2"})
	if err != nil {
		t.Fatal(err)
	}
	ip, err = a.Allocate("id-should-fail-2")
	if err == nil {
		t.Errorf("unexpected success %s", ip.String())
	}
}

func TestAllocatorV6(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "temp")
	if err != nil {
		t.Errorf("create tempDir: %v", err)
	}
	t.Logf("logs on %s", tempDir)
	dbDir = tempDir
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	// initialize variables
	err = start()
	if err != nil {
		t.Fatal(err)
	}
	logger = log.New(os.Stdout, "", 0) // 0 flag for no timestamps/prefixes

	a, err := NewAllocator(netip.MustParsePrefix("2001:db8::/64"))
	if err != nil {
		t.Fatal(err)
	}
	// can not allocate on the reserved space
	err = writeNetworkConfigWithoutIPs(&NetworkConfig{ContainerID: "id-static-1"})
	if err != nil {
		t.Fatal(err)
	}
	err = a.AllocateAddress(netip.MustParseAddr("2001:db8::2"), "id-static-1")
	if err == nil {
		t.Fatal("can not alllocate on the reserved space")
	}

	// let's try some allocations
	var i uint64
	for i = 0; i < 100; i++ {
		id := fmt.Sprintf("id-%d", i)
		err = writeNetworkConfigWithoutIPs(&NetworkConfig{ContainerID: id})
		if err != nil {
			t.Fatal(err)
		}
		_, err := a.Allocate(id)
		if err != nil {
			t.Fatal(err)
		}
	}
	// release and allocate manually
	a.Release(netip.MustParseAddr("2001:db8::aa"))
	err = writeNetworkConfigWithoutIPs(&NetworkConfig{ContainerID: "id-static-2"})
	if err != nil {
		t.Fatal(err)
	}
	err = a.AllocateAddress(netip.MustParseAddr("2001:db8::aa"), "id-static-2")
	if err != nil {
		t.Fatal(err)
	}

	// it should fail to allocate since it is alreadya allocated
	err = writeNetworkConfigWithoutIPs(&NetworkConfig{ContainerID: "id-static-3"})
	if err != nil {
		t.Fatal(err)
	}
	err = a.AllocateAddress(netip.MustParseAddr("2001:db8::aa"), "id-static-3")
	if err == nil {
		t.Fatalf("unexpected success for IP 2001:db8::aa")
	}

}

func Test_broadcastAddress(t *testing.T) {
	tests := []struct {
		name   string
		subnet netip.Prefix
		want   netip.Addr
	}{
		{
			name:   "ipv4",
			subnet: netip.MustParsePrefix("192.168.0.0/24"),
			want:   netip.MustParseAddr("192.168.0.255"),
		},
		{
			name:   "ipv4 no nibble boundary",
			subnet: netip.MustParsePrefix("10.0.0.0/12"),
			want:   netip.MustParseAddr("10.15.255.255"),
		},
		{
			name:   "ipv6",
			subnet: netip.MustParsePrefix("fd00:1:2:3::/64"),
			want:   netip.MustParseAddr("fd00:1:2:3:FFFF:FFFF:FFFF:FFFF"),
		},
		{
			name:   "ipv6 00fc::/112",
			subnet: netip.MustParsePrefix("00fc::/112"),
			want:   netip.MustParseAddr("fc::ffff"),
		},
		{
			name:   "ipv6 fc00::/112",
			subnet: netip.MustParsePrefix("fc00::/112"),
			want:   netip.MustParseAddr("fc00::ffff"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := broadcastAddress(tt.subnet); !reflect.DeepEqual(got, tt.want) || err != nil {
				t.Errorf("broadcastAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_addOffsetAddress(t *testing.T) {
	tests := []struct {
		name    string
		address netip.Addr
		offset  uint64
		want    netip.Addr
	}{
		{
			name:    "IPv4 offset 0",
			address: netip.MustParseAddr("192.168.0.0"),
			offset:  0,
			want:    netip.MustParseAddr("192.168.0.0"),
		},
		{
			name:    "IPv4 offset 0 not nibble boundary",
			address: netip.MustParseAddr("192.168.0.11"),
			offset:  0,
			want:    netip.MustParseAddr("192.168.0.11"),
		},
		{
			name:    "IPv4 offset 1",
			address: netip.MustParseAddr("192.168.0.0"),
			offset:  1,
			want:    netip.MustParseAddr("192.168.0.1"),
		},
		{
			name:    "IPv4 offset 1 not nibble boundary",
			address: netip.MustParseAddr("192.168.0.11"),
			offset:  1,
			want:    netip.MustParseAddr("192.168.0.12"),
		},
		{
			name:    "IPv6 offset 1",
			address: netip.MustParseAddr("fd00:1:2:3::"),
			offset:  1,
			want:    netip.MustParseAddr("fd00:1:2:3::1"),
		},
		{
			name:    "IPv6 offset 1 not nibble boundary",
			address: netip.MustParseAddr("fd00:1:2:3::a"),
			offset:  1,
			want:    netip.MustParseAddr("fd00:1:2:3::b"),
		},
		{
			name:    "IPv4 offset last",
			address: netip.MustParseAddr("192.168.0.0"),
			offset:  255,
			want:    netip.MustParseAddr("192.168.0.255"),
		},
		{
			name:    "IPv6 offset last",
			address: netip.MustParseAddr("fd00:1:2:3::"),
			offset:  0x7FFFFFFFFFFFFFFF,
			want:    netip.MustParseAddr("fd00:1:2:3:7FFF:FFFF:FFFF:FFFF"),
		},
		{
			name:    "IPv4 offset middle",
			address: netip.MustParseAddr("192.168.0.0"),
			offset:  128,
			want:    netip.MustParseAddr("192.168.0.128"),
		},
		{
			name:    "IPv4 with leading zeros",
			address: netip.MustParseAddr("0.0.1.8"),
			offset:  138,
			want:    netip.MustParseAddr("0.0.1.146"),
		},
		{
			name:    "IPv6 with leading zeros",
			address: netip.MustParseAddr("00fc::1"),
			offset:  255,
			want:    netip.MustParseAddr("fc::100"),
		},
		{
			name:    "IPv6 offset 255",
			address: netip.MustParseAddr("2001:db8:1::101"),
			offset:  255,
			want:    netip.MustParseAddr("2001:db8:1::200"),
		},
		{
			name:    "IPv6 offset 1025",
			address: netip.MustParseAddr("fd00:1:2:3::"),
			offset:  1025,
			want:    netip.MustParseAddr("fd00:1:2:3::401"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := addOffsetAddress(tt.address, tt.offset)
			if !reflect.DeepEqual(got, tt.want) || err != nil {
				t.Errorf("offsetAddress() = %v, want %v", got, tt.want)
			}
			// double check to avoid mistakes on the hardcoded values
			// avoid large numbers or it will timeout the test
			if tt.offset < 2048 {
				want := tt.address
				var i uint64
				for i = 0; i < tt.offset; i++ {
					want = want.Next()
				}
				if !reflect.DeepEqual(got, tt.want) || err != nil {
					t.Errorf("offsetAddress() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
