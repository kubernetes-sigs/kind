// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gps

import (
	"bytes"
	"fmt"
)

type errorSlice []error

func (errs errorSlice) Error() string {
	var buf bytes.Buffer
	fmt.Fprintln(&buf)
	for i, err := range errs {
		fmt.Fprintf(&buf, "\t(%d) %s\n", i+1, err)
	}
	return buf.String()
}

func (errs errorSlice) Format(f fmt.State, c rune) {
	fmt.Fprintln(f)
	for i, err := range errs {
		if ferr, ok := err.(fmt.Formatter); ok {
			fmt.Fprintf(f, "\t(%d) ", i+1)
			ferr.Format(f, c)
			fmt.Fprint(f, "\n")
		} else {
			fmt.Fprintf(f, "\t(%d) %s\n", i+1, err)
		}
	}
}
