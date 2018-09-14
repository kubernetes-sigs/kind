// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"runtime"

	"github.com/golang/dep"
)

var (
	version    = "devel"
	buildDate  string
	commitHash string
)

const versionHelp = `Show the dep version information`

func (cmd *versionCommand) Name() string { return "version" }
func (cmd *versionCommand) Args() string {
	return ""
}
func (cmd *versionCommand) ShortHelp() string { return versionHelp }
func (cmd *versionCommand) LongHelp() string  { return versionHelp }
func (cmd *versionCommand) Hidden() bool      { return false }

func (cmd *versionCommand) Register(fs *flag.FlagSet) {}

type versionCommand struct{}

func (cmd *versionCommand) Run(ctx *dep.Ctx, args []string) error {
	ctx.Out.Printf(`dep:
 version     : %s
 build date  : %s
 git hash    : %s
 go version  : %s
 go compiler : %s
 platform    : %s/%s
 features    : ImportDuringSolve=%v
`, version, buildDate, commitHash,
		runtime.Version(), runtime.Compiler, runtime.GOOS, runtime.GOARCH,
		importDuringSolve())
	return nil
}
