// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gps

import (
	"fmt"
	"sort"
)

// Lock represents data from a lock file (or however the implementing tool
// chooses to store it) at a particular version that is relevant to the
// satisfiability solving process.
//
// In general, the information produced by gps on finding a successful
// solution is all that would be necessary to constitute a lock file, though
// tools can include whatever other information they want in their storage.
type Lock interface {
	// Projects returns the list of LockedProjects contained in the lock data.
	Projects() []LockedProject

	// The set of imports (and required statements) that were the inputs that
	// generated this Lock. It is acceptable to return a nil slice from this
	// method if the information cannot reasonably be made available.
	InputImports() []string
}

// sortLockedProjects returns a sorted copy of lps, or itself if already sorted.
func sortLockedProjects(lps []LockedProject) []LockedProject {
	if len(lps) <= 1 || sort.SliceIsSorted(lps, func(i, j int) bool {
		return lps[i].Ident().Less(lps[j].Ident())
	}) {
		return lps
	}
	cp := make([]LockedProject, len(lps))
	copy(cp, lps)
	sort.Slice(cp, func(i, j int) bool {
		return cp[i].Ident().Less(cp[j].Ident())
	})
	return cp
}

// LockedProject is a single project entry from a lock file. It expresses the
// project's name, one or both of version and underlying revision, the network
// URI for accessing it, the path at which it should be placed within a vendor
// directory, and the packages that are used in it.
type LockedProject interface {
	Ident() ProjectIdentifier
	Version() Version
	Packages() []string
	Eq(LockedProject) bool
	String() string
}

// lockedProject is the default implementation of LockedProject.
type lockedProject struct {
	pi   ProjectIdentifier
	v    UnpairedVersion
	r    Revision
	pkgs []string
}

// SimpleLock is a helper for tools to easily describe lock data when they know
// that input imports are unavailable.
type SimpleLock []LockedProject

var _ Lock = SimpleLock{}

// Projects returns the entire contents of the SimpleLock.
func (l SimpleLock) Projects() []LockedProject {
	return l
}

// InputImports returns a nil string slice, as SimpleLock does not provide a way
// of capturing string slices.
func (l SimpleLock) InputImports() []string {
	return nil
}

// NewLockedProject creates a new LockedProject struct with a given
// ProjectIdentifier (name and optional upstream source URL), version. and list
// of packages required from the project.
//
// Note that passing a nil version will cause a panic. This is a correctness
// measure to ensure that the solver is never exposed to a version-less lock
// entry. Such a case would be meaningless - the solver would have no choice but
// to simply dismiss that project. By creating a hard failure case via panic
// instead, we are trying to avoid inflicting the resulting pain on the user by
// instead forcing a decision on the Analyzer implementation.
func NewLockedProject(id ProjectIdentifier, v Version, pkgs []string) LockedProject {
	if v == nil {
		panic("must provide a non-nil version to create a LockedProject")
	}

	lp := lockedProject{
		pi:   id,
		pkgs: pkgs,
	}

	switch tv := v.(type) {
	case Revision:
		lp.r = tv
	case branchVersion:
		lp.v = tv
	case semVersion:
		lp.v = tv
	case plainVersion:
		lp.v = tv
	case versionPair:
		lp.r = tv.r
		lp.v = tv.v
	}

	return lp
}

// Ident returns the identifier describing the project. This includes both the
// local name (the root name by which the project is referenced in import paths)
// and the network name, where the upstream source lives.
func (lp lockedProject) Ident() ProjectIdentifier {
	return lp.pi
}

// Version assembles together whatever version and/or revision data is
// available into a single Version.
func (lp lockedProject) Version() Version {
	if lp.r == "" {
		return lp.v
	}

	if lp.v == nil {
		return lp.r
	}

	return lp.v.Pair(lp.r)
}

// Eq checks if two LockedProject instances are equal. The implementation
// assumes both Packages lists are already sorted lexicographically.
func (lp lockedProject) Eq(lp2 LockedProject) bool {
	if lp.pi != lp2.Ident() {
		return false
	}

	var uv UnpairedVersion
	switch tv := lp2.Version().(type) {
	case Revision:
		if lp.r != tv {
			return false
		}
	case versionPair:
		if lp.r != tv.r {
			return false
		}
		uv = tv.v
	case branchVersion, semVersion, plainVersion:
		// For now, we're going to say that revisions must be present in order
		// to indicate equality. We may need to change this later, as it may be
		// more appropriate to enforce elsewhere.
		return false
	}

	v1n := lp.v == nil
	v2n := uv == nil

	if v1n != v2n {
		return false
	}

	if !v1n && !lp.v.Matches(uv) {
		return false
	}

	opkgs := lp2.Packages()
	if len(lp.pkgs) != len(opkgs) {
		return false
	}

	for k, v := range lp.pkgs {
		if opkgs[k] != v {
			return false
		}
	}

	return true
}

// Packages returns the list of packages from within the LockedProject that are
// actually used in the import graph. Some caveats:
//
//  * The names given are relative to the root import path for the project. If
//    the root package itself is imported, it's represented as ".".
//  * Just because a package path isn't included in this list doesn't mean it's
//    safe to remove - it could contain C files, or other assets, that can't be
//    safely removed.
//  * The slice is not a copy. If you need to modify it, copy it first.
func (lp lockedProject) Packages() []string {
	return lp.pkgs
}

func (lp lockedProject) String() string {
	return fmt.Sprintf("%s@%s with packages: %v",
		lp.Ident(), lp.Version(), lp.pkgs)
}

type safeLock struct {
	p []LockedProject
	i []string
}

func (sl safeLock) InputImports() []string {
	return sl.i
}

func (sl safeLock) Projects() []LockedProject {
	return sl.p
}

// prepLock ensures a lock is prepared and safe for use by the solver. This is
// mostly about defensively ensuring that no outside routine can modify the lock
// while the solver is in-flight.
//
// This is achieved by copying the lock's data into a new safeLock.
func prepLock(l Lock) safeLock {
	pl := l.Projects()

	rl := safeLock{
		p: make([]LockedProject, len(pl)),
	}
	copy(rl.p, pl)

	rl.i = make([]string, len(l.InputImports()))
	copy(rl.i, l.InputImports())

	return rl
}
