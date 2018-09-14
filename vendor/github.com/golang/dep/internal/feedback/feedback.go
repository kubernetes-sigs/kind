// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package feedback

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/golang/dep/gps"
)

const (
	// ConsTypeConstraint represents a constraint
	ConsTypeConstraint = "constraint"

	// ConsTypeHint represents a constraint type hint
	ConsTypeHint = "hint"

	// DepTypeDirect represents a direct dependency
	DepTypeDirect = "direct dep"

	// DepTypeTransitive represents a transitive dependency,
	// or a dependency of a dependency
	DepTypeTransitive = "transitive dep"

	// DepTypeImported represents a dependency imported by an external tool
	DepTypeImported = "imported dep"
)

// ConstraintFeedback holds project constraint feedback data
type ConstraintFeedback struct {
	Constraint, LockedVersion, Revision, ConstraintType, DependencyType, ProjectPath string
}

// NewConstraintFeedback builds a feedback entry for a constraint in the manifest.
func NewConstraintFeedback(pc gps.ProjectConstraint, depType string) *ConstraintFeedback {
	cf := &ConstraintFeedback{
		Constraint:     pc.Constraint.String(),
		ProjectPath:    string(pc.Ident.ProjectRoot),
		DependencyType: depType,
	}

	if _, ok := pc.Constraint.(gps.Revision); ok {
		cf.ConstraintType = ConsTypeHint
	} else {
		cf.ConstraintType = ConsTypeConstraint
	}

	return cf
}

// NewLockedProjectFeedback builds a feedback entry for a project in the lock.
func NewLockedProjectFeedback(lp gps.LockedProject, depType string) *ConstraintFeedback {
	cf := &ConstraintFeedback{
		ProjectPath:    string(lp.Ident().ProjectRoot),
		DependencyType: depType,
	}

	switch vt := lp.Version().(type) {
	case gps.PairedVersion:
		cf.LockedVersion = vt.String()
		cf.Revision = vt.Revision().String()
	case gps.UnpairedVersion: // Logically this should never occur, but handle for completeness sake
		cf.LockedVersion = vt.String()
	case gps.Revision:
		cf.Revision = vt.String()
	}

	return cf
}

// LogFeedback logs feedback on changes made to the manifest or lock.
func (cf ConstraintFeedback) LogFeedback(logger *log.Logger) {
	if cf.Constraint != "" {
		logger.Printf("  %v", GetUsingFeedback(cf.Constraint, cf.ConstraintType, cf.DependencyType, cf.ProjectPath))
	}
	if cf.Revision != "" {
		logger.Printf("  %v", GetLockingFeedback(cf.LockedVersion, cf.Revision, cf.DependencyType, cf.ProjectPath))
	}
}

type brokenImport interface {
	String() string
}

type modifiedImport struct {
	source, branch, revision, version *StringDiff
	projectPath                       string
}

func (mi modifiedImport) String() string {
	var pv string
	var pr string
	pp := mi.projectPath

	var cr string
	var cv string
	cp := ""

	if mi.revision != nil {
		pr = fmt.Sprintf("(%s)", trimSHA(mi.revision.Previous))
		cr = fmt.Sprintf("(%s)", trimSHA(mi.revision.Current))
	}

	if mi.version != nil {
		pv = mi.version.Previous
		cv = mi.version.Current
	} else if mi.branch != nil {
		pv = mi.branch.Previous
		cv = mi.branch.Current
	}

	if mi.source != nil {
		pp = fmt.Sprintf("%s(%s)", mi.projectPath, mi.source.Previous)
		cp = fmt.Sprintf(" for %s(%s)", mi.projectPath, mi.source.Current)
	}

	// Warning: Unable to preserve imported lock VERSION/BRANCH (REV) for PROJECT(SOURCE). Locking in VERSION/BRANCH (REV) for PROJECT(SOURCE)
	return fmt.Sprintf("%v %s for %s. Locking in %v %s%s", pv, pr, pp, cv, cr, cp)
}

type removedImport struct {
	source, branch, revision, version *StringDiff
	projectPath                       string
}

func (ri removedImport) String() string {
	var pr string
	var pv string
	pp := ri.projectPath

	if ri.revision != nil {
		pr = fmt.Sprintf("(%s)", trimSHA(ri.revision.Previous))
	}

	if ri.version != nil {
		pv = ri.version.Previous
	} else if ri.branch != nil {
		pv = ri.branch.Previous
	}

	if ri.source != nil {
		pp = fmt.Sprintf("%s(%s)", ri.projectPath, ri.source.Previous)
	}

	// Warning: Unable to preserve imported lock VERSION/BRANCH (REV) for PROJECT(SOURCE). Locking in VERSION/BRANCH (REV) for PROJECT(SOURCE)
	return fmt.Sprintf("%v %s for %s. The project was removed from the lock because it is not used.", pv, pr, pp)
}

// BrokenImportFeedback holds information on changes to locks pre- and post- solving.
type BrokenImportFeedback struct {
	brokenImports []brokenImport
}

// NewBrokenImportFeedback builds a feedback entry that compares an initially
// imported, unsolved lock to the same lock after it has been solved.
func NewBrokenImportFeedback(ld *LockDiff) *BrokenImportFeedback {
	bi := &BrokenImportFeedback{}
	if ld == nil {
		return bi
	}

	for _, lpd := range ld.Modify {
		if lpd.Branch == nil && lpd.Revision == nil && lpd.Source == nil && lpd.Version == nil {
			continue
		}
		bi.brokenImports = append(bi.brokenImports, modifiedImport{
			projectPath: string(lpd.Name),
			source:      lpd.Source,
			branch:      lpd.Branch,
			revision:    lpd.Revision,
			version:     lpd.Version,
		})
	}

	for _, lpd := range ld.Remove {
		bi.brokenImports = append(bi.brokenImports, removedImport{
			projectPath: string(lpd.Name),
			source:      lpd.Source,
			branch:      lpd.Branch,
			revision:    lpd.Revision,
			version:     lpd.Version,
		})
	}

	return bi
}

// LogFeedback logs a warning for all changes between the initially imported and post- solve locks
func (b BrokenImportFeedback) LogFeedback(logger *log.Logger) {
	for _, bi := range b.brokenImports {
		logger.Printf("Warning: Unable to preserve imported lock %v\n", bi)
	}
}

// GetUsingFeedback returns a dependency "using" feedback message. For example:
//
//    Using ^1.0.0 as constraint for direct dep github.com/foo/bar
//    Using 1b8edb3 as hint for direct dep github.com/bar/baz
func GetUsingFeedback(version, consType, depType, projectPath string) string {
	if depType == DepTypeImported {
		return fmt.Sprintf("Using %s as initial %s for %s %s", version, consType, depType, projectPath)
	}
	return fmt.Sprintf("Using %s as %s for %s %s", version, consType, depType, projectPath)
}

// GetLockingFeedback returns a dependency "locking" feedback message. For
// example:
//
//    Locking in v1.1.4 (bc29b4f) for direct dep github.com/foo/bar
//    Locking in master (436f39d) for transitive dep github.com/baz/qux
func GetLockingFeedback(version, revision, depType, projectPath string) string {
	revision = trimSHA(revision)

	if depType == DepTypeImported {
		if version == "" {
			version = "*"
		}
		return fmt.Sprintf("Trying %s (%s) as initial lock for %s %s", version, revision, depType, projectPath)
	}
	return fmt.Sprintf("Locking in %s (%s) for %s %s", version, revision, depType, projectPath)
}

// trimSHA checks if revision is a valid SHA1 digest and trims to 7 characters.
func trimSHA(revision string) string {
	if len(revision) == 40 {
		if _, err := hex.DecodeString(revision); err == nil {
			// Valid SHA1 digest
			revision = revision[0:7]
		}
	}

	return revision
}
