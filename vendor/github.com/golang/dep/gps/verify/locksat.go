// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package verify

import (
	radix "github.com/armon/go-radix"
	"github.com/golang/dep/gps"
	"github.com/golang/dep/gps/paths"
	"github.com/golang/dep/gps/pkgtree"
)

// LockSatisfaction holds the compound result of LockSatisfiesInputs, allowing
// the caller to inspect each of several orthogonal possible types of failure.
//
// The zero value assumes that there was no input lock, which necessarily means
// the inputs were not satisfied. This zero value means we err on the side of
// failure.
type LockSatisfaction struct {
	// If LockExisted is false, it indicates that a nil gps.Lock was passed to
	// LockSatisfiesInputs().
	LockExisted bool
	// MissingImports is the set of import paths that were present in the
	// inputs but missing in the Lock.
	MissingImports []string
	// ExcessImports is the set of import paths that were present in the Lock
	// but absent from the inputs.
	ExcessImports []string
	// UnmatchedConstraints reports any normal, non-override constraint rules that
	// were not satisfied by the corresponding LockedProject in the Lock.
	UnmetConstraints map[gps.ProjectRoot]ConstraintMismatch
	// UnmatchedOverrides reports any override rules that were not satisfied by the
	// corresponding LockedProject in the Lock.
	UnmetOverrides map[gps.ProjectRoot]ConstraintMismatch
}

// ConstraintMismatch is a two-tuple of a gps.Version, and a gps.Constraint that
// does not allow that version.
type ConstraintMismatch struct {
	C gps.Constraint
	V gps.Version
}

// LockSatisfiesInputs determines whether the provided Lock satisfies all the
// requirements indicated by the inputs (RootManifest and PackageTree).
//
// The second parameter is expected to be the list of imports that were used to
// generate the input Lock. Without this explicit list, it is not possible to
// compute package imports that may have been removed. Figuring out that
// negative space would require exploring the entire graph to ensure there are
// no in-edges for particular imports.
func LockSatisfiesInputs(l gps.Lock, m gps.RootManifest, ptree pkgtree.PackageTree) LockSatisfaction {
	if l == nil {
		return LockSatisfaction{}
	}

	lsat := LockSatisfaction{
		LockExisted:      true,
		UnmetOverrides:   make(map[gps.ProjectRoot]ConstraintMismatch),
		UnmetConstraints: make(map[gps.ProjectRoot]ConstraintMismatch),
	}

	var ig *pkgtree.IgnoredRuleset
	var req map[string]bool
	if m != nil {
		ig = m.IgnoredPackages()
		req = m.RequiredPackages()
	}

	rm, _ := ptree.ToReachMap(true, true, false, ig)
	reach := rm.FlattenFn(paths.IsStandardImportPath)

	inlock := make(map[string]bool, len(l.InputImports()))
	ininputs := make(map[string]bool, len(reach)+len(req))

	type lockUnsatisfy uint8
	const (
		missingFromLock lockUnsatisfy = iota
		inAdditionToLock
	)

	pkgDiff := make(map[string]lockUnsatisfy)

	for _, imp := range reach {
		ininputs[imp] = true
	}

	for imp := range req {
		ininputs[imp] = true
	}

	for _, imp := range l.InputImports() {
		inlock[imp] = true
	}

	for ip := range ininputs {
		if !inlock[ip] {
			pkgDiff[ip] = missingFromLock
		} else {
			// So we don't have to revisit it below
			delete(inlock, ip)
		}
	}

	// Something in the missing list might already be in the packages list,
	// because another package in the depgraph imports it. We could make a
	// special case for that, but it would break the simplicity of the model and
	// complicate the notion of LockSatisfaction.Passed(), so let's see if we
	// can get away without it.

	for ip := range inlock {
		if !ininputs[ip] {
			pkgDiff[ip] = inAdditionToLock
		}
	}

	for ip, typ := range pkgDiff {
		if typ == missingFromLock {
			lsat.MissingImports = append(lsat.MissingImports, ip)
		} else {
			lsat.ExcessImports = append(lsat.ExcessImports, ip)
		}
	}

	eff := findEffectualConstraints(m, ininputs)
	ovr, constraints := m.Overrides(), m.DependencyConstraints()

	for _, lp := range l.Projects() {
		pr := lp.Ident().ProjectRoot

		if pp, has := ovr[pr]; has {
			if !pp.Constraint.Matches(lp.Version()) {
				lsat.UnmetOverrides[pr] = ConstraintMismatch{
					C: pp.Constraint,
					V: lp.Version(),
				}
			}
			// The constraint isn't considered if we have an override,
			// independent of whether the override is satisfied.
			continue
		}

		if pp, has := constraints[pr]; has && eff[string(pr)] && !pp.Constraint.Matches(lp.Version()) {
			lsat.UnmetConstraints[pr] = ConstraintMismatch{
				C: pp.Constraint,
				V: lp.Version(),
			}
		}
	}

	return lsat
}

// Satisfied is a shortcut method that indicates whether there were any ways in
// which the Lock did not satisfy the inputs. It will return true only if the
// Lock was satisfactory in all respects vis-a-vis the inputs.
func (ls LockSatisfaction) Satisfied() bool {
	if !ls.LockExisted {
		return false
	}

	if len(ls.MissingImports) > 0 {
		return false
	}

	if len(ls.ExcessImports) > 0 {
		return false
	}

	if len(ls.UnmetOverrides) > 0 {
		return false
	}

	if len(ls.UnmetConstraints) > 0 {
		return false
	}

	return true
}

func findEffectualConstraints(m gps.Manifest, imports map[string]bool) map[string]bool {
	eff := make(map[string]bool)
	xt := radix.New()

	for pr := range m.DependencyConstraints() {
		// FIXME(sdboyer) this has the trailing slash ambiguity problem; adapt
		// code from the solver
		xt.Insert(string(pr), nil)
	}

	for imp := range imports {
		if root, _, has := xt.LongestPrefix(imp); has {
			eff[root] = true
		}
	}

	return eff
}
