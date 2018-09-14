// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package verify

import (
	"github.com/golang/dep/gps"
)

// VerifiableProject composes a LockedProject to indicate what the hash digest
// of a file tree for that LockedProject should be, given the PruneOptions and
// the list of packages.
type VerifiableProject struct {
	gps.LockedProject
	PruneOpts gps.PruneOptions
	Digest    VersionedDigest
}
