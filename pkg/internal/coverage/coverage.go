// +build !coverage

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

package coverage

import (
	"flag"
	"testing"
)

// RunMainWithCoverage should be only once in a special TestMain to invoke
// an app with coverage instrumentation
// This binary must be built with go test -c -covermode -coverpkg ...
func RunMainWithCoverage(main func(), outputPath string) {
	defer writeCoverage(outputPath)
	main()
}

func writeCoverage(outputPath string) {
	// We're not actually going to run any tests, but we need Go to think we did so it writes
	// coverage information to disk. To achieve this, we create a bunch of empty test suites and
	// have it "run" them.
	tests := []testing.InternalTest{}
	benchmarks := []testing.InternalBenchmark{}
	examples := []testing.InternalExample{}

	var deps fakeTestDeps

	_ = flag.CommandLine.Lookup("test.coverprofile").Value.Set(outputPath)
	dummyRun := testing.MainStart(deps, tests, benchmarks, examples)
	dummyRun.Run()
}
