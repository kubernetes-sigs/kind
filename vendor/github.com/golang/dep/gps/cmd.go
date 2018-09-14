// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gps

import (
	"os"
)

func (c cmd) Args() []string {
	return c.Cmd.Args
}

func (c cmd) SetDir(dir string) {
	c.Cmd.Dir = dir
}

func (c cmd) SetEnv(env []string) {
	c.Cmd.Env = env
}

func init() {
	// For our git repositories, we very much assume a "regular" topology.
	// Therefore, no value for the following variables can be relevant to
	// us. Unsetting globally properly propagates to libraries like
	// github.com/Masterminds/vcs, which cannot make the same assumption in
	// general.
	parasiteGitVars := []string{"GIT_DIR", "GIT_INDEX_FILE", "GIT_OBJECT_DIRECTORY", "GIT_WORK_TREE"}
	for _, e := range parasiteGitVars {
		os.Unsetenv(e)
	}
}
