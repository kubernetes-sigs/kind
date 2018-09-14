// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/golang/dep"
	"github.com/golang/dep/gps"
	"github.com/golang/dep/gps/verify"
	"github.com/pkg/errors"
)

const checkShortHelp = `Check if imports, Gopkg.toml, and Gopkg.lock are in sync`
const checkLongHelp = `
Check determines if your project is in a good state. If problems are found, it
prints a description of each issue, then exits 1. Passing -q suppresses output.

Flags control which specific checks will be run. By default, dep check verifies
that Gopkg.lock is in sync with Gopkg.toml and the imports in your project's .go
files, and that the vendor directory is in sync with Gopkg.lock. These checks
can be disabled with -skip-lock and -skip-vendor, respectively.

(See https://golang.github.io/dep/docs/ensure-mechanics.html#staying-in-sync for
more information on what it means to be "in sync.")

If your workflow necessitates that you modify the contents of vendor, you can
force check to ignore hash mismatches on a per-project basis by naming
project roots in Gopkg.toml's "noverify" list.
`

type checkCommand struct {
	quiet                bool
	skiplock, skipvendor bool
}

func (cmd *checkCommand) Name() string { return "check" }
func (cmd *checkCommand) Args() string {
	return "[-q] [-skip-lock] [-skip-vendor]"
}
func (cmd *checkCommand) ShortHelp() string { return checkShortHelp }
func (cmd *checkCommand) LongHelp() string  { return checkLongHelp }
func (cmd *checkCommand) Hidden() bool      { return false }

func (cmd *checkCommand) Register(fs *flag.FlagSet) {
	fs.BoolVar(&cmd.skiplock, "skip-lock", false, "Skip checking that imports and Gopkg.toml are in sync with Gopkg.lock")
	fs.BoolVar(&cmd.skipvendor, "skip-vendor", false, "Skip checking that vendor is in sync with Gopkg.lock")
	fs.BoolVar(&cmd.quiet, "q", false, "Suppress non-error output")
}

func (cmd *checkCommand) Run(ctx *dep.Ctx, args []string) error {
	logger := ctx.Out
	if cmd.quiet {
		logger = log.New(ioutil.Discard, "", 0)
	}

	p, err := ctx.LoadProject()
	if err != nil {
		return err
	}

	sm, err := ctx.SourceManager()
	if err != nil {
		return err
	}

	sm.UseDefaultSignalHandling()
	defer sm.Release()

	var fail bool
	if !cmd.skiplock {
		if p.Lock == nil {
			return errors.New("Gopkg.lock does not exist, cannot check it against imports and Gopkg.toml")
		}

		lsat := verify.LockSatisfiesInputs(p.Lock, p.Manifest, p.RootPackageTree)
		delta := verify.DiffLocks(p.Lock, p.ChangedLock)
		sat, changed := lsat.Satisfied(), delta.Changed(verify.PruneOptsChanged|verify.HashVersionChanged)

		if changed || !sat {
			fail = true
			logger.Println("# Gopkg.lock is out of sync:")
			if !sat {
				logger.Printf("%s\n", sprintLockUnsat(lsat))
			}
			if changed {
				// Sort, for deterministic output.
				var ordered []string
				for pr := range delta.ProjectDeltas {
					ordered = append(ordered, string(pr))
				}
				sort.Strings(ordered)

				for _, pr := range ordered {
					lpd := delta.ProjectDeltas[gps.ProjectRoot(pr)]
					// Only two possible changes right now are prune opts
					// changing or a missing hash digest (for old Gopkg.lock
					// files)
					if lpd.PruneOptsChanged() {
						// Override what's on the lockdiff with the extra info we have;
						// this lets us excise PruneNestedVendorDirs and get the real
						// value from the input param in place.
						old := lpd.PruneOptsBefore & ^gps.PruneNestedVendorDirs
						new := lpd.PruneOptsAfter & ^gps.PruneNestedVendorDirs
						logger.Printf("%s: prune options changed (%s -> %s)\n", pr, old, new)
					}
					if lpd.HashVersionWasZero() {
						logger.Printf("%s: no hash digest in lock\n", pr)
					}
				}
			}
		}
	}

	if !cmd.skipvendor {
		if p.Lock == nil {
			return errors.New("Gopkg.lock does not exist, cannot check vendor against it")
		}

		statuses, err := p.VerifyVendor()
		if err != nil {
			return errors.Wrap(err, "error while verifying vendor")
		}

		if fail {
			logger.Println()
		}

		noverify := make(map[string]bool)
		for _, skip := range p.Manifest.NoVerify {
			noverify[skip] = true
		}

		var vendorfail bool
		// One full pass through, to see if we need to print the header, and to
		// create an array of names to sort for deterministic output.
		var ordered []string
		for path, status := range statuses {
			ordered = append(ordered, path)

			switch status {
			case verify.DigestMismatchInLock, verify.HashVersionMismatch, verify.EmptyDigestInLock:
				// NoVerify applies only to these three cases.
				if noverify[path] {
					continue
				}
				fallthrough
			case verify.NotInTree, verify.NotInLock:
				fail = true
				if !vendorfail {
					vendorfail = true
					logger.Println("# vendor is out of sync:")
				}

			}
		}
		sort.Strings(ordered)

		for _, pr := range ordered {
			var nvSuffix string
			if noverify[pr] {
				nvSuffix = "  (CHECK IGNORED: marked noverify in Gopkg.toml)"
			}

			status := statuses[pr]
			switch status {
			case verify.NotInTree:
				logger.Printf("%s: missing from vendor\n", pr)
			case verify.NotInLock:
				fi, err := os.Stat(filepath.Join(p.AbsRoot, "vendor", pr))
				if err != nil {
					return errors.Wrap(err, "could not stat file that VerifyVendor claimed existed")
				}
				if fi.IsDir() {
					logger.Printf("%s: unused project\n", pr)
				} else {
					logger.Printf("%s: orphaned file\n", pr)
				}
			case verify.DigestMismatchInLock:
				logger.Printf("%s: hash of vendored tree not equal to digest in Gopkg.lock%s\n", pr, nvSuffix)
			case verify.EmptyDigestInLock:
				logger.Printf("%s: no digest in Gopkg.lock to compare against hash of vendored tree%s\n", pr, nvSuffix)
			case verify.HashVersionMismatch:
				// This will double-print if the hash version is zero, but
				// that's a rare case that really only occurs before the first
				// run with a version of dep >=0.5.0, so it's fine.
				logger.Printf("%s: hash algorithm mismatch, want version %v%s\n", pr, verify.HashVersion, nvSuffix)
			}
		}
	}

	if fail {
		return silentfail{}
	}
	return nil
}

func sprintLockUnsat(lsat verify.LockSatisfaction) string {
	var buf bytes.Buffer
	sort.Strings(lsat.MissingImports)
	for _, missing := range lsat.MissingImports {
		fmt.Fprintf(&buf, "%s: imported or required, but missing from Gopkg.lock's input-imports\n", missing)
	}

	sort.Strings(lsat.ExcessImports)
	for _, excess := range lsat.ExcessImports {
		fmt.Fprintf(&buf, "%s: in Gopkg.lock's input-imports, but neither imported nor required\n", excess)
	}

	var ordered []string
	for pr := range lsat.UnmetOverrides {
		ordered = append(ordered, string(pr))
	}
	sort.Strings(ordered)
	for _, pr := range ordered {
		unmatched := lsat.UnmetOverrides[gps.ProjectRoot(pr)]
		fmt.Fprintf(&buf, "%s@%s: not allowed by override %s\n", pr, unmatched.V, unmatched.C)
	}

	ordered = ordered[:0]
	for pr := range lsat.UnmetConstraints {
		ordered = append(ordered, string(pr))
	}
	sort.Strings(ordered)
	for _, pr := range ordered {
		unmatched := lsat.UnmetConstraints[gps.ProjectRoot(pr)]
		fmt.Fprintf(&buf, "%s@%s: not allowed by constraint %s\n", pr, unmatched.V, unmatched.C)
	}
	return strings.TrimSpace(buf.String())
}
