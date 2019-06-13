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

package concurrent

import (
	"errors"
	"sync"
	"time"

	"sigs.k8s.io/kind/pkg/util"
)

const timeout = 60 * time.Second

// UntilError runs all funcs in separate goroutines, returning the
// first non-nil error returned from funcs, or nil if all funcs return nil
func UntilError(funcs []func() error) error {
	errCh := make(chan error, len(funcs))
	for _, f := range funcs {
		f := f // capture f
		go func() {
			errCh <- f()
		}()
	}
	for i := 0; i < len(funcs); i++ {
		select {
		case err := <-errCh:
			if err != nil {
				return err
			}
		case <-time.After(timeout):
			return errors.New("UntilError: Timeout waiting for concurrent function")
		}
	}
	return nil
}

// Coalesce runs fns concurrently, returning an Errors if there are > 1 errors
func Coalesce(fns ...func() error) error {
	// run all fns concurrently
	ch := make(chan error, len(fns))
	var wg sync.WaitGroup
	for _, fn := range fns {
		wg.Add(1)
		go func(f func() error) {
			defer wg.Done()
			ch <- f()
		}(fn)
	}
	if waitTimeout(&wg, timeout) {
		return errors.New("Coalesce: Timeout waiting for wait group")
	}
	close(ch)
	// collect up and return errors
	errs := []error{}
	for err := range ch {
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 1 {
		return util.Flatten(errs)
	} else if len(errs) == 1 {
		return errs[0]
	}
	return nil
}

// waitTimeout waits for the waitgroup for the specified max timeout.
// Returns true if waiting timed out.
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}
