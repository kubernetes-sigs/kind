// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"strconv"
)

const (
	flagImportDuringSolveKey = "ImportDuringSolve"
)

var (
	flagImportDuringSolve = "false"
)

var featureFlags = map[string]bool{
	flagImportDuringSolveKey: parseFeatureFlag(flagImportDuringSolve),
}

func parseFeatureFlag(flag string) bool {
	flagValue, _ := strconv.ParseBool(flag)
	return flagValue
}

func readFeatureFlag(flag string) (bool, error) {
	if flagValue, ok := featureFlags[flag]; ok {
		return flagValue, nil
	}

	return false, fmt.Errorf("undefined feature flag: %s", flag)
}

func importDuringSolve() bool {
	return featureFlags[flagImportDuringSolveKey]
}
