/*
Copyright 2018 The Kubernetes Authors.

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

/*
This implements our image entrypoint. It:
- waits for SIGUSR1
- then execs (argv[1], argv[1:], env[:])

This allows us to perform other actions in the "node" container via `docker exec`
_before_ we have actually "booted" the init and everything else along with it.

We can then send SIGUSR1 to this process to trigger starting the "actual"
entrypoint when we are done preforming any provisioning on the "node".

NOTE: this is implemented as a single go1.X file, using only the stdlib.
This makes it easier to build portably, and is all we need for what it does.
*/

// Entrypoint implements a small docker image entrypoint that waits for SIGUSR1
// before execing os.Args[1:]
package main

import (
	"os"
	"os/signal"
	"syscall"

	"log"
)

// yes this should be the c macro, but on linux in docker you're going to get this anyhow
// http://man7.org/linux/man-pages/man7/signal.7.html
// https://github.com/moby/moby/blob/562df8c2d6f48601c8d1df7256389569d25c0bf1/pkg/signal/signal_linux.go#L10
const sigrtmin = 34

func main() {
	// prevent zombie processes since we will be PID1 for a while
	// https://linux.die.net/man/2/waitpid
	signal.Ignore(syscall.SIGCHLD)

	// grab the "real" entrypoint command and args from our args
	if len(os.Args) < 2 {
		log.Fatal("Not enough arguments to entrypoint!")
	}
	cmd, argv := os.Args[1], os.Args[1:]

	// wait for SIGUSR1 (or exit on SIGRTMIN+3 to match systemd)
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR1, syscall.Signal(sigrtmin+3))
	log.Println("Waiting for SIGUSR1 ...")
	sig := <-c
	if sig != syscall.SIGUSR1 {
		log.Printf("Exiting after signal: %v != SIGUSR1", sig)
		return
	}

	// then exec to the "real" entrypoint, keeping the env
	log.Printf("Received SIGUSR1, execing to: %v %v\n", cmd, argv)
	syscall.Exec(cmd, argv, os.Environ())
}
