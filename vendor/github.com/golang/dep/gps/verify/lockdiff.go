// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package verify

import (
	"bytes"
	"sort"
	"strings"

	"github.com/golang/dep/gps"
)

// DeltaDimension defines a bitset enumerating all of the different dimensions
// along which a Lock, and its constitutent components, can change.
type DeltaDimension uint32

// Each flag represents an ortohgonal dimension along which Locks can vary with
// respect to each other.
const (
	InputImportsChanged DeltaDimension = 1 << iota
	ProjectAdded
	ProjectRemoved
	SourceChanged
	VersionChanged
	RevisionChanged
	PackagesChanged
	PruneOptsChanged
	HashVersionChanged
	HashChanged
	AnyChanged = (1 << iota) - 1
)

// LockDelta represents all possible differences between two Locks.
type LockDelta struct {
	AddedImportInputs   []string
	RemovedImportInputs []string
	ProjectDeltas       map[gps.ProjectRoot]LockedProjectDelta
}

// LockedProjectDelta represents all possible state changes of a LockedProject
// within a Lock. It encapsulates the property-level differences represented by
// a LockedProjectPropertiesDelta, but can also represent existence deltas - a
// given name came to exist, or cease to exist, across two Locks.
type LockedProjectDelta struct {
	Name                         gps.ProjectRoot
	ProjectRemoved, ProjectAdded bool
	LockedProjectPropertiesDelta
}

// LockedProjectPropertiesDelta represents all possible differences between the
// properties of two LockedProjects. It can represent deltas for
// VerifiableProject properties, as well.
type LockedProjectPropertiesDelta struct {
	PackagesAdded, PackagesRemoved      []string
	VersionBefore, VersionAfter         gps.UnpairedVersion
	RevisionBefore, RevisionAfter       gps.Revision
	SourceBefore, SourceAfter           string
	PruneOptsBefore, PruneOptsAfter     gps.PruneOptions
	HashVersionBefore, HashVersionAfter int
	HashChanged                         bool
}

// DiffLocks compares two locks and computes a semantically rich delta between
// them.
func DiffLocks(l1, l2 gps.Lock) LockDelta {
	// Default nil locks to empty locks, so that we can still generate a diff.
	if l1 == nil {
		if l2 == nil {
			// But both locks being nil results in an empty delta.
			return LockDelta{}
		}
		l1 = gps.SimpleLock{}
	}
	if l2 == nil {
		l2 = gps.SimpleLock{}
	}

	p1, p2 := l1.Projects(), l2.Projects()

	p1 = sortLockedProjects(p1)
	p2 = sortLockedProjects(p2)

	diff := LockDelta{
		ProjectDeltas: make(map[gps.ProjectRoot]LockedProjectDelta),
	}

	var i2next int
	for i1 := 0; i1 < len(p1); i1++ {
		lp1 := p1[i1]
		pr1 := lp1.Ident().ProjectRoot

		lpd := LockedProjectDelta{
			Name: pr1,
		}

		for i2 := i2next; i2 < len(p2); i2++ {
			lp2 := p2[i2]
			pr2 := lp2.Ident().ProjectRoot

			switch strings.Compare(string(pr1), string(pr2)) {
			case 0: // Found a matching project
				lpd.LockedProjectPropertiesDelta = DiffLockedProjectProperties(lp1, lp2)
				i2next = i2 + 1 // Don't visit this project again
			case +1: // Found a new project
				diff.ProjectDeltas[pr2] = LockedProjectDelta{
					Name:         pr2,
					ProjectAdded: true,
				}
				i2next = i2 + 1 // Don't visit this project again
				continue        // Keep looking for a matching project
			case -1: // Project has been removed, handled below
				lpd.ProjectRemoved = true
			}

			break // Done evaluating this project, move onto the next
		}

		diff.ProjectDeltas[pr1] = lpd
	}

	// Anything that still hasn't been evaluated are adds
	for i2 := i2next; i2 < len(p2); i2++ {
		lp2 := p2[i2]
		pr2 := lp2.Ident().ProjectRoot
		diff.ProjectDeltas[pr2] = LockedProjectDelta{
			Name:         pr2,
			ProjectAdded: true,
		}
	}

	diff.AddedImportInputs, diff.RemovedImportInputs = findAddedAndRemoved(l1.InputImports(), l2.InputImports())

	return diff
}

func findAddedAndRemoved(l1, l2 []string) (add, remove []string) {
	// Computing package add/removes might be optimizable to O(n) (?), but it's
	// not critical path for any known case, so not worth the effort right now.
	p1, p2 := make(map[string]bool, len(l1)), make(map[string]bool, len(l2))

	for _, pkg := range l1 {
		p1[pkg] = true
	}
	for _, pkg := range l2 {
		p2[pkg] = true
	}

	for pkg := range p1 {
		if !p2[pkg] {
			remove = append(remove, pkg)
		}
	}
	for pkg := range p2 {
		if !p1[pkg] {
			add = append(add, pkg)
		}
	}

	return add, remove
}

// DiffLockedProjectProperties takes two gps.LockedProject and computes a delta
// for each of their component properties.
//
// This function is focused exclusively on the properties of a LockedProject. As
// such, it does not compare the ProjectRoot part of the LockedProject's
// ProjectIdentifier, as those are names, and the concern here is a difference
// in properties, not intrinsic identity.
func DiffLockedProjectProperties(lp1, lp2 gps.LockedProject) LockedProjectPropertiesDelta {
	ld := LockedProjectPropertiesDelta{
		SourceBefore: lp1.Ident().Source,
		SourceAfter:  lp2.Ident().Source,
	}

	ld.PackagesAdded, ld.PackagesRemoved = findAddedAndRemoved(lp1.Packages(), lp2.Packages())

	switch v := lp1.Version().(type) {
	case gps.PairedVersion:
		ld.VersionBefore, ld.RevisionBefore = v.Unpair(), v.Revision()
	case gps.Revision:
		ld.RevisionBefore = v
	case gps.UnpairedVersion:
		// This should ideally never happen
		ld.VersionBefore = v
	}

	switch v := lp2.Version().(type) {
	case gps.PairedVersion:
		ld.VersionAfter, ld.RevisionAfter = v.Unpair(), v.Revision()
	case gps.Revision:
		ld.RevisionAfter = v
	case gps.UnpairedVersion:
		// This should ideally never happen
		ld.VersionAfter = v
	}

	vp1, ok1 := lp1.(VerifiableProject)
	vp2, ok2 := lp2.(VerifiableProject)

	if ok1 && ok2 {
		ld.PruneOptsBefore, ld.PruneOptsAfter = vp1.PruneOpts, vp2.PruneOpts
		ld.HashVersionBefore, ld.HashVersionAfter = vp1.Digest.HashVersion, vp2.Digest.HashVersion

		if !bytes.Equal(vp1.Digest.Digest, vp2.Digest.Digest) {
			ld.HashChanged = true
		}
	} else if ok1 {
		ld.PruneOptsBefore = vp1.PruneOpts
		ld.HashVersionBefore = vp1.Digest.HashVersion
		ld.HashChanged = true
	} else if ok2 {
		ld.PruneOptsAfter = vp2.PruneOpts
		ld.HashVersionAfter = vp2.Digest.HashVersion
		ld.HashChanged = true
	}

	return ld
}

// Changed indicates whether the delta contains a change along the dimensions
// with their corresponding bits set.
//
// This implementation checks the topmost-level Lock properties
func (ld LockDelta) Changed(dims DeltaDimension) bool {
	if dims&InputImportsChanged != 0 && (len(ld.AddedImportInputs) > 0 || len(ld.RemovedImportInputs) > 0) {
		return true
	}

	for _, ld := range ld.ProjectDeltas {
		if ld.Changed(dims & ^InputImportsChanged) {
			return true
		}
	}

	return false
}

// Changes returns a bitset indicating the dimensions along which deltas exist across
// all contents of the LockDelta.
//
// This recurses down into the individual LockedProjectDeltas contained within
// the LockDelta. A single delta along a particular dimension from a single
// project is sufficient to flip the bit on for that dimension.
func (ld LockDelta) Changes() DeltaDimension {
	var dd DeltaDimension
	if len(ld.AddedImportInputs) > 0 || len(ld.RemovedImportInputs) > 0 {
		dd |= InputImportsChanged
	}

	for _, ld := range ld.ProjectDeltas {
		dd |= ld.Changes()
	}

	return dd
}

// Changed indicates whether the delta contains a change along the dimensions
// with their corresponding bits set.
//
// For example, if only the Revision changed, and this method is called with
// SourceChanged | VersionChanged, it will return false; if it is called with
// VersionChanged | RevisionChanged, it will return true.
func (ld LockedProjectDelta) Changed(dims DeltaDimension) bool {
	if dims&ProjectAdded != 0 && ld.WasAdded() {
		return true
	}

	if dims&ProjectRemoved != 0 && ld.WasRemoved() {
		return true
	}

	return ld.LockedProjectPropertiesDelta.Changed(dims & ^ProjectAdded & ^ProjectRemoved)
}

// Changes returns a bitset indicating the dimensions along which there were
// changes between the compared LockedProjects. This includes both
// existence-level deltas (add/remove) and property-level deltas.
func (ld LockedProjectDelta) Changes() DeltaDimension {
	var dd DeltaDimension
	if ld.WasAdded() {
		dd |= ProjectAdded
	}

	if ld.WasRemoved() {
		dd |= ProjectRemoved
	}

	return dd | ld.LockedProjectPropertiesDelta.Changes()
}

// WasRemoved returns true if the named project existed in the first lock, but
// did not exist in the second lock.
func (ld LockedProjectDelta) WasRemoved() bool {
	return ld.ProjectRemoved
}

// WasAdded returns true if the named project did not exist in the first lock,
// but did exist in the second lock.
func (ld LockedProjectDelta) WasAdded() bool {
	return ld.ProjectAdded
}

// Changed indicates whether the delta contains a change along the dimensions
// with their corresponding bits set.
//
// For example, if only the Revision changed, and this method is called with
// SourceChanged | VersionChanged, it will return false; if it is called with
// VersionChanged | RevisionChanged, it will return true.
func (ld LockedProjectPropertiesDelta) Changed(dims DeltaDimension) bool {
	if dims&SourceChanged != 0 && ld.SourceChanged() {
		return true
	}
	if dims&RevisionChanged != 0 && ld.RevisionChanged() {
		return true
	}
	if dims&PruneOptsChanged != 0 && ld.PruneOptsChanged() {
		return true
	}
	if dims&HashChanged != 0 && ld.HashChanged {
		return true
	}
	if dims&HashVersionChanged != 0 && ld.HashVersionChanged() {
		return true
	}
	if dims&VersionChanged != 0 && ld.VersionChanged() {
		return true
	}
	if dims&PackagesChanged != 0 && ld.PackagesChanged() {
		return true
	}

	return false
}

// Changes returns a bitset indicating the dimensions along which there were
// changes between the compared LockedProjects.
func (ld LockedProjectPropertiesDelta) Changes() DeltaDimension {
	var dd DeltaDimension
	if ld.SourceChanged() {
		dd |= SourceChanged
	}
	if ld.RevisionChanged() {
		dd |= RevisionChanged
	}
	if ld.PruneOptsChanged() {
		dd |= PruneOptsChanged
	}
	if ld.HashChanged {
		dd |= HashChanged
	}
	if ld.HashVersionChanged() {
		dd |= HashVersionChanged
	}
	if ld.VersionChanged() {
		dd |= VersionChanged
	}
	if ld.PackagesChanged() {
		dd |= PackagesChanged
	}

	return dd
}

// SourceChanged returns true if the source field differed between the first and
// second locks.
func (ld LockedProjectPropertiesDelta) SourceChanged() bool {
	return ld.SourceBefore != ld.SourceAfter
}

// VersionChanged returns true if the version property differed between the
// first and second locks. In addition to simple changes (e.g. 1.0.1 -> 1.0.2),
// this also includes all possible version type changes either going from a
// paired version to a plain revision, or the reverse direction, or the type of
// unpaired version changing (e.g. branch -> semver).
func (ld LockedProjectPropertiesDelta) VersionChanged() bool {
	if ld.VersionBefore == nil && ld.VersionAfter == nil {
		return false
	} else if (ld.VersionBefore == nil || ld.VersionAfter == nil) || (ld.VersionBefore.Type() != ld.VersionAfter.Type()) {
		return true
	} else if !ld.VersionBefore.Matches(ld.VersionAfter) {
		return true
	}

	return false
}

// RevisionChanged returns true if the revision property differed between the
// first and second locks.
func (ld LockedProjectPropertiesDelta) RevisionChanged() bool {
	return ld.RevisionBefore != ld.RevisionAfter
}

// PackagesChanged returns true if the package set gained or lost members (or
// both) between the first and second locks.
func (ld LockedProjectPropertiesDelta) PackagesChanged() bool {
	return len(ld.PackagesAdded) > 0 || len(ld.PackagesRemoved) > 0
}

// PruneOptsChanged returns true if the pruning flags for the project changed
// between the first and second locks.
func (ld LockedProjectPropertiesDelta) PruneOptsChanged() bool {
	return ld.PruneOptsBefore != ld.PruneOptsAfter
}

// HashVersionChanged returns true if the version of the hashing algorithm
// changed between the first and second locks.
func (ld LockedProjectPropertiesDelta) HashVersionChanged() bool {
	return ld.HashVersionBefore != ld.HashVersionAfter
}

// HashVersionWasZero returns true if the first lock had a zero hash version,
// which can only mean it was uninitialized.
func (ld LockedProjectPropertiesDelta) HashVersionWasZero() bool {
	return ld.HashVersionBefore == 0
}

// sortLockedProjects returns a sorted copy of lps, or itself if already sorted.
func sortLockedProjects(lps []gps.LockedProject) []gps.LockedProject {
	if len(lps) <= 1 || sort.SliceIsSorted(lps, func(i, j int) bool {
		return lps[i].Ident().Less(lps[j].Ident())
	}) {
		return lps
	}

	cp := make([]gps.LockedProject, len(lps))
	copy(cp, lps)

	sort.Slice(cp, func(i, j int) bool {
		return cp[i].Ident().Less(cp[j].Ident())
	})
	return cp
}
