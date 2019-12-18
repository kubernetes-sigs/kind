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

package exec

import "io"

// FakeCmder implements a very simple Cmder fake that returns a copy of
// the FakeCmd each time
// WARNING: this API is not yet stable! It will change!
// TODO(bentheelder): design a more robust exec fake
type FakeCmder struct {
	FakeCmd
}

// FakeCmd implements a very simple Cmd fake
// WARNING: this API is not yet stable! It will change!
// TODO(bentheelder): design a more robust exec fake
type FakeCmd struct {
	Out    []byte
	ErrOut []byte
	Error  error
	Stdout io.Writer
	Stderr io.Writer
}

// Command implements Cmder.Command
func (f *FakeCmder) Command(name string, arg ...string) Cmd {
	f2 := f.FakeCmd
	return &f2
}

// Run implements Cmd.Run
func (f *FakeCmd) Run() error {
	if f.Stdout != nil {
		// TODO: how should the fake handle these errors?
		_, _ = f.Stdout.Write(f.Out)
	}
	if f.Stderr != nil {
		_, _ = f.Stderr.Write(f.ErrOut)
	}
	return f.Error
}

// SetEnv "implements" Cmd.SetEnv but discards the input
func (f *FakeCmd) SetEnv(e ...string) Cmd { return f }

// SetStdin "implements" Cmd.SetStdin but discards the input
func (f *FakeCmd) SetStdin(r io.Reader) Cmd { return f }

// SetStdout implements Cmd.SetStdout
func (f *FakeCmd) SetStdout(s io.Writer) Cmd {
	f.Stdout = s
	return f
}

// SetStderr implements Cmd.SetStderr
func (f *FakeCmd) SetStderr(s io.Writer) Cmd {
	f.Stderr = s
	return f
}
