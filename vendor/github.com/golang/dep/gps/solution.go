// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gps

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// A Solution is returned by a solver run. It is mostly just a Lock, with some
// additional methods that report information about the solve run.
type Solution interface {
	Lock
	// The name of the ProjectAnalyzer used in generating this solution.
	AnalyzerName() string
	// The version of the ProjectAnalyzer used in generating this solution.
	AnalyzerVersion() int
	// The name of the Solver used in generating this solution.
	SolverName() string
	// The version of the Solver used in generating this solution.
	SolverVersion() int
	Attempts() int
}

type solution struct {
	// The projects selected by the solver.
	p []LockedProject

	// The import inputs that created this solution (including requires).
	i []string

	// The number of solutions that were attempted
	att int

	// The analyzer info
	analyzerInfo ProjectAnalyzerInfo

	// The solver used in producing this solution
	solv Solver
}

// WriteProgress informs about the progress of WriteDepTree.
type WriteProgress struct {
	Count   int
	Total   int
	LP      LockedProject
	Failure bool
}

func (p WriteProgress) String() string {
	msg := "Wrote"
	if p.Failure {
		msg = "Failed to write"
	}
	return fmt.Sprintf("(%d/%d) %s %s@%s", p.Count, p.Total, msg, p.LP.Ident(), p.LP.Version())
}

const concurrentWriters = 16

// WriteDepTree takes a basedir, a Lock and a RootPruneOptions and exports all
// the projects listed in the lock to the appropriate target location within basedir.
//
// If the goal is to populate a vendor directory, basedir should be the absolute
// path to that vendor directory, not its parent (a project root, typically).
//
// It requires a SourceManager to do the work. Prune options are read from the
// passed manifest.
//
// If onWrite is not nil, it will be called after each project write. Calls are ordered and atomic.
func WriteDepTree(basedir string, l Lock, sm SourceManager, co CascadingPruneOptions, onWrite func(WriteProgress)) error {
	if l == nil {
		return fmt.Errorf("must provide non-nil Lock to WriteDepTree")
	}

	if err := os.MkdirAll(basedir, 0777); err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(context.TODO())
	lps := l.Projects()
	sem := make(chan struct{}, concurrentWriters)
	var cnt struct {
		sync.Mutex
		i int
	}

	for i := range lps {
		p := lps[i] // per-iteration copy

		g.Go(func() error {
			err := func() error {
				select {
				case sem <- struct{}{}:
					defer func() { <-sem }()
				case <-ctx.Done():
					return ctx.Err()
				}

				ident := p.Ident()
				projectRoot := string(ident.ProjectRoot)
				to := filepath.FromSlash(filepath.Join(basedir, projectRoot))

				if err := sm.ExportProject(ctx, ident, p.Version(), to); err != nil {
					return errors.Wrapf(err, "failed to export %s", projectRoot)
				}

				err := PruneProject(to, p, co.PruneOptionsFor(ident.ProjectRoot))
				if err != nil {
					return errors.Wrapf(err, "failed to prune %s", projectRoot)
				}

				return ctx.Err()
			}()

			switch err {
			case context.Canceled, context.DeadlineExceeded:
				// Don't report "secondary" errors.
			default:
				if onWrite != nil {
					// Increment and call atomically to prevent re-ordering.
					cnt.Lock()
					cnt.i++
					onWrite(WriteProgress{
						Count:   cnt.i,
						Total:   len(lps),
						LP:      p,
						Failure: err != nil,
					})
					cnt.Unlock()
				}
			}

			return err
		})
	}

	err := g.Wait()
	if err != nil {
		os.RemoveAll(basedir)
	}
	return errors.Wrap(err, "failed to write dep tree")
}

func (r solution) Projects() []LockedProject {
	return r.p
}

func (r solution) InputImports() []string {
	return r.i
}

func (r solution) Attempts() int {
	return r.att
}

func (r solution) AnalyzerName() string {
	return r.analyzerInfo.Name
}

func (r solution) AnalyzerVersion() int {
	return r.analyzerInfo.Version
}

func (r solution) SolverName() string {
	return r.solv.Name()
}

func (r solution) SolverVersion() int {
	return r.solv.Version()
}
