//go:build tools
// +build tools

/*
Package site is used to track binary dependencies with go modules
https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
Namely: hugo
*/
package site

import (
	_ "github.com/gohugoio/hugo"
)
