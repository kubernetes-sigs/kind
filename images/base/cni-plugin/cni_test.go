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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/vishvananda/netns"
)

// avoid problems with shared state
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestCNIPlugin(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("Test requires root privileges.")
	}
	tests := []struct {
		name   string
		ranges []netip.Prefix
	}{
		{"ipv4", []netip.Prefix{netip.MustParsePrefix("10.0.0.0/24")}},
		{"ipv6", []netip.Prefix{netip.MustParsePrefix("2001:db8::/64")}},
		{"dual", []netip.Prefix{netip.MustParsePrefix("10.0.0.0/24"), netip.MustParsePrefix("2001:db8::/64")}},
	}

	for _, rt := range tests {
		t.Run(rt.name, func(t *testing.T) {
			null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
			if err != nil {
				t.Fatal(err)
			}
			os.Stdout = null
			tempDir, err := ioutil.TempDir("", "temp")
			if err != nil {
				t.Errorf("create tempDir: %v", err)
			}
			t.Logf("logs on %s", tempDir)
			dbDir = tempDir
			t.Setenv("CNI_LOG_FILE", filepath.Join(tempDir, "test.log"))
			t.Cleanup(func() { os.RemoveAll(tempDir) })

			// initialize variables
			err = start()
			if err != nil {
				t.Fatal(err)
			}
			// 1. Prepare test environment
			// Save the current network namespace
			origns, err := netns.Get()
			if err != nil {
				t.Fatalf("unexpected error trying to get namespace: %v", err)
			}
			defer origns.Close()

			rndString := make([]byte, 4)
			_, err = rand.Read(rndString)
			if err != nil {
				t.Errorf("fail to generate random name: %v", err)
			}
			nsName := fmt.Sprintf("ns%x", rndString)
			testNS, err := netns.NewNamed(nsName)
			if err != nil {
				t.Fatalf("Failed to create network namespace: %v", err)
			}
			defer netns.DeleteNamed(nsName)
			defer testNS.Close()

			// Switch back to the original namespace
			netns.Set(origns)

			// 3. Prepare CNI configuration
			cniConfig := KindnetConf{
				NetConf: types.NetConf{
					CNIVersion: "0.3.1",
					Name:       "test-network",
					Type:       "cni-kindnet",
				},
			}
			for _, cidr := range rt.ranges {
				cniConfig.Ranges = append(cniConfig.Ranges, cidr.String())
			}

			data, err := json.Marshal(cniConfig)
			if err != nil {
				t.Fatalf("Failed to serialize cni config: %v", err)
			}
			//  Prepare CNI arguments
			args := &skel.CmdArgs{
				ContainerID: "test-container",
				Netns:       filepath.Join("/run/netns/", nsName),
				IfName:      "eth0",
				StdinData:   data,
			}

			//  Execute ADD command
			if err := cmdAdd(args); err != nil {
				t.Fatalf("CNI ADD command failed: %v", err)
			}

			// check connectivity from the namespace
			func() {
				runtime.LockOSThread()
				defer runtime.UnlockOSThread()
				err := netns.Set(testNS)
				if err != nil {
					t.Fatal(err)
				}
				// use the first IP family first network address
				cmd := exec.Command("ping", "-c", "3", rt.ranges[0].Masked().Addr().String())

				// (Optional) Get output
				_, err = cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("no connectivity from namespace: %v", err)
				}

				// Switch back to the original namespace
				err = netns.Set(origns)
				if err != nil {
					t.Fatal(err)
				}
			}()

			//  Execute DEL command
			if err := cmdDel(args); err != nil {
				t.Errorf("CNI DEL command failed: %v", err)
			}

			// TODO test check
			err = cmdCheck(args)
		})
	}
}

func TestAddDel(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("Test requires root privileges.")
	}
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = null
	now := time.Now()
	total := 300
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

	// Save the current network namespace
	origns, err := netns.Get()
	if err != nil {
		t.Fatalf("unexpected error trying to get namespace: %v", err)
	}
	defer origns.Close()

	var successes atomic.Uint64
	var wg sync.WaitGroup
	for i := 0; i < total; i++ {
		time.Sleep(10 * time.Millisecond) // avoid flakiness due to races with netlink
		runtime.LockOSThread()
		rndString := make([]byte, 4)
		_, err := rand.Read(rndString)
		if err != nil {
			t.Errorf("fail to generate random name: %v", err)
		}
		nsName := fmt.Sprintf("ns%x", rndString)
		testNS, err := netns.NewNamed(nsName)
		if err != nil {
			t.Errorf("Failed to create network namespace: %v", err)
		}
		// Switch back to the original namespace
		netns.Set(origns)
		runtime.UnlockOSThread()

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer netns.DeleteNamed(nsName)
			defer testNS.Close()

			success := true
			// Prepare CNI configuration
			cniConfig := KindnetConf{
				NetConf: types.NetConf{
					CNIVersion: "0.3.1",
					Name:       "test-network",
					Type:       "cni-kindnet",
				},
				Ranges: []string{"169.254.99.0/24"},
			}

			data, err := json.Marshal(cniConfig)
			if err != nil {
				success = false
				t.Errorf("Failed to serialize cni config: %v", err)
			}
			//  Prepare CNI arguments
			args := &skel.CmdArgs{
				ContainerID: nsName,
				Netns:       filepath.Join("/run/netns/", nsName),
				IfName:      "eth0",
				StdinData:   data,
			}

			//  Execute ADD command
			if err := cmdAdd(args); err != nil {
				success = false
				t.Errorf("CNI ADD command failed: %v", err)
				out, _ := exec.Command("ip", "link").CombinedOutput()
				t.Logf("ip link: %s", string(out))
			}

			//  Execute DEL command
			if err := cmdDel(args); err != nil {
				success = false
				t.Errorf("CNI DEL command failed: %v", err)
			}
			if success {
				successes.Add(1)
			}
		}()
	}
	// database should be empty at this point
	wg.Wait()
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM pods").Scan(&count)
	if err != nil {
		log.Fatal(err)
	}

	// Check if the count is zero
	if count != 0 {
		t.Errorf("The pods database is not empty: %d", count)
	} else {
		t.Logf("no pods remaining in the db")
	}
	t.Logf("%d success out of %d in %v", successes.Load(), total, time.Since(now))
}

func TestAdds(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("Test requires root privileges.")
	}
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = null

	now := time.Now()
	total := 300
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

	// Save the current network namespace
	origns, err := netns.Get()
	if err != nil {
		t.Fatalf("unexpected error trying to get namespace: %v", err)
	}
	defer origns.Close()

	var successes atomic.Uint64
	var wg sync.WaitGroup
	argsCh := make(chan *skel.CmdArgs, total)
	for i := 0; i < total; i++ {
		time.Sleep(10 * time.Millisecond) // avoid flakiness due to races with netlink
		runtime.LockOSThread()
		rndString := make([]byte, 4)
		_, err := rand.Read(rndString)
		if err != nil {
			t.Errorf("fail to generate random name: %v", err)
		}
		nsName := fmt.Sprintf("ns%x", rndString)
		_, err = netns.NewNamed(nsName)
		if err != nil {
			t.Errorf("Failed to create network namespace: %v", err)
		}
		// Switch back to the original namespace
		netns.Set(origns)
		runtime.UnlockOSThread()

		wg.Add(1)
		go func() {
			defer wg.Done()

			success := true
			// Prepare CNI configuration
			cniConfig := KindnetConf{
				NetConf: types.NetConf{
					CNIVersion: "0.3.1",
					Name:       "test-network",
					Type:       "cni-kindnet",
				},
				Ranges: []string{"fd00:1:2:3::/64"},
			}

			data, err := json.Marshal(cniConfig)
			if err != nil {
				success = false
				t.Errorf("Failed to serialize cni config: %v", err)
			}
			//  Prepare CNI arguments
			args := &skel.CmdArgs{
				ContainerID: nsName,
				Netns:       filepath.Join("/run/netns/", nsName),
				IfName:      "eth0",
				StdinData:   data,
			}

			//  Execute ADD command
			if err := cmdAdd(args); err != nil {
				success = false
				t.Errorf("CNI ADD command failed: %v", err)
			}

			if success {
				successes.Add(1)
			}
			argsCh <- args
		}()
	}
	// database should be empty at this point
	wg.Wait()
	t.Logf("%d success added out of %d in %v", successes.Load(), total, time.Since(now))

	for i := 0; i < total; i++ {
		args := <-argsCh
		wg.Add(1)
		go func() {
			defer wg.Done()
			//  Execute DEL command
			if err := cmdDel(args); err != nil {
				t.Errorf("CNI DEL command failed: %v", err)
			}
			netns.DeleteNamed(filepath.Base(args.Netns))
		}()
	}

	wg.Wait()
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM pods").Scan(&count)
	if err != nil {
		log.Fatal(err)
	}

	// Check if the count is zero
	if count != 0 {
		t.Errorf("The pods database is not empty: %d", count)
	} else {
		t.Logf("no pods remaining in the db")
	}
	t.Logf("%d success out of %d in %v", successes.Load(), total, time.Since(now))
}

func TestHostPort(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("Test requires root privileges.")
	}
	containerPort := 8080
	tests := []struct {
		name      string
		ranges    []netip.Prefix
		hostports []PortMapEntry
	}{
		{
			name:   "ipv4-localhost",
			ranges: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/24")},
			hostports: []PortMapEntry{{
				HostPort:      18090,
				ContainerPort: containerPort,
				Protocol:      "TCP",
				HostIP:        "127.0.0.1",
			}},
		},
		{
			name:   "ipv4",
			ranges: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/24")},
			hostports: []PortMapEntry{{
				HostPort:      18090,
				ContainerPort: containerPort,
				Protocol:      "TCP",
				HostIP:        "",
			}},
		},
		{
			name:   "ipv6-localhost",
			ranges: []netip.Prefix{netip.MustParsePrefix("2001:db8::/64")},
			hostports: []PortMapEntry{{
				HostPort:      18090,
				ContainerPort: containerPort,
				Protocol:      "TCP",
				HostIP:        "::1",
			}},
		},
		{
			name:   "ipv6",
			ranges: []netip.Prefix{netip.MustParsePrefix("2001:db8::/64")},
			hostports: []PortMapEntry{{
				HostPort:      18090,
				ContainerPort: containerPort,
				Protocol:      "TCP",
				HostIP:        "",
			}},
		},
	}

	for _, rt := range tests {
		t.Run(rt.name, func(t *testing.T) {
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

			// 1. Prepare test environment
			// Save the current network namespace
			runtime.LockOSThread()
			origns, err := netns.Get()
			if err != nil {
				t.Fatalf("unexpected error trying to get namespace: %v", err)
			}
			defer origns.Close()

			rndString := make([]byte, 4)
			_, err = rand.Read(rndString)
			if err != nil {
				t.Errorf("fail to generate random name: %v", err)
			}
			nsName := fmt.Sprintf("ns%x", rndString)
			testNS, err := netns.NewNamed(nsName)
			if err != nil {
				t.Fatalf("Failed to create network namespace: %v", err)
			}
			defer netns.DeleteNamed(nsName)
			defer testNS.Close()

			// open a listener inside the namespace
			lnAt, err := net.Listen("tcp", fmt.Sprintf(":%d", containerPort))
			if err != nil {
				t.Fatalf("Failed to create listener on network namespace: %v", err)
			}
			// Switch back to the original namespace
			netns.Set(origns)
			runtime.UnlockOSThread()

			mux := http.NewServeMux()
			mux.HandleFunc("/cni-test", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("Connection from: " + r.RemoteAddr))
			})
			go func() {
				http.Serve(lnAt, mux)
			}()

			// 3. Prepare CNI configuration
			cniConfig := KindnetConf{
				NetConf: types.NetConf{
					CNIVersion: "0.3.1",
					Name:       "test-network",
					Type:       "cni-kindnet",
				},
			}
			for _, cidr := range rt.ranges {
				cniConfig.Ranges = append(cniConfig.Ranges, cidr.String())
			}
			for _, hostport := range rt.hostports {
				cniConfig.RuntimeConfig.PortMaps = append(cniConfig.RuntimeConfig.PortMaps, hostport)
			}

			data, err := json.Marshal(cniConfig)
			if err != nil {
				t.Fatalf("Failed to serialize cni config: %v", err)
			}
			//  Prepare CNI arguments
			args := &skel.CmdArgs{
				ContainerID: "test-container",
				Netns:       filepath.Join("/run/netns/", nsName),
				IfName:      "eth0",
				StdinData:   data,
			}

			//  Execute ADD command
			if err := cmdAdd(args); err != nil {
				t.Fatalf("CNI ADD command failed: %v", err)
			}

			cmd := exec.Command("nft", "list", "table", "inet", "cni-kindnet")
			_, err = cmd.CombinedOutput()
			if err != nil {
				t.Errorf("no table cni-kindnet on namespace: %v", err)
			}
			// t.Logf("rules after ADD: %s", string(out))

			// we need to dial from the host otherwise we need to set route_localnet on IPv4 and does not work for IPv6
			// Use the first address from the range since is the one added by the CNI command to the container interface on the host to be used as default gw.
			dialer := &net.Dialer{
				LocalAddr: &net.TCPAddr{
					IP:   rt.ranges[0].Addr().AsSlice(),
					Port: 0,
				},
			}
			client := http.Client{Transport: &http.Transport{
				Dial: dialer.Dial,
			}}

			// Test connectivity
			for _, hostport := range rt.hostports {
				// TODO: connect to any IP in the host for the unspecified
				requestURL := fmt.Sprintf("http://localhost:%d/cni-test", hostport.HostPort)
				res, err := client.Get(requestURL)
				if err != nil {
					t.Fatalf("error making http request: %v\n", err)
				}
				resBody, err := io.ReadAll(res.Body)
				if err != nil {
					t.Fatalf("could not read response body: %v\n", err)
				}
				// t.Logf("client: response body: %s\n", string(resBody))
				if !strings.Contains(string(resBody), "Connection from") {
					t.Fatalf("unexpected response body: %s\n", string(resBody))
				}
			}

			//  Execute DEL command
			if err := cmdDel(args); err != nil {
				t.Errorf("CNI DEL command failed: %v", err)
			}

			cmd = exec.Command("nft", "list", "table", "inet", "cni-kindnet")
			_, err = cmd.CombinedOutput()
			if err != nil {
				t.Errorf("no connectivity from namespace: %v", err)
			}
			// t.Logf("rules after DEL: %s", string(out))

			// TODO test check
			err = cmdCheck(args)
		})
	}
}
