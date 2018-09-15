// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gps

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/Masterminds/vcs"
)

// A maybeSource represents a set of information that, given some
// typically-expensive network effort, could be transformed into a proper source.
//
// Wrapping these up as their own type achieves two goals:
//
// * Allows control over when deduction logic triggers network activity
// * Makes it easy to attempt multiple URLs for a given import path
type maybeSource interface {
	// try tries to set up a source.
	try(ctx context.Context, cachedir string) (source, error)
	URL() *url.URL
	fmt.Stringer
}

type maybeSources []maybeSource

func (mbs maybeSources) possibleURLs() []*url.URL {
	urlslice := make([]*url.URL, len(mbs))
	for i, mb := range mbs {
		urlslice[i] = mb.URL()
	}
	return urlslice
}

// sourceCachePath returns a url-sanitized source cache dir path.
func sourceCachePath(cacheDir, sourceURL string) string {
	return filepath.Join(cacheDir, "sources", sanitizer.Replace(sourceURL))
}

type maybeGitSource struct {
	url *url.URL
}

func (m maybeGitSource) try(ctx context.Context, cachedir string) (source, error) {
	ustr := m.url.String()
	path := sourceCachePath(cachedir, ustr)

	r, err := vcs.NewGitRepo(ustr, path)
	if err != nil {
		os.RemoveAll(path)
		r, err = vcs.NewGitRepo(ustr, path)
		if err != nil {
			return nil, unwrapVcsErr(err)
		}
	}

	return &gitSource{
		baseVCSSource: baseVCSSource{
			repo: &gitRepo{r},
		},
	}, nil
}

func (m maybeGitSource) URL() *url.URL {
	return m.url
}

func (m maybeGitSource) String() string {
	return fmt.Sprintf("%T: %s", m, ufmt(m.url))
}

type maybeGopkginSource struct {
	// the original gopkg.in import path. this is used to create the on-disk
	// location to avoid duplicate resource management - e.g., if instances of
	// a gopkg.in project are accessed via different schemes, or if the
	// underlying github repository is accessed directly.
	opath string
	// the actual upstream URL - always github
	url *url.URL
	// the major version to apply for filtering
	major uint64
	// whether or not the source package is "unstable"
	unstable bool
}

func (m maybeGopkginSource) try(ctx context.Context, cachedir string) (source, error) {
	// We don't actually need a fully consistent transform into the on-disk path
	// - just something that's unique to the particular gopkg.in domain context.
	// So, it's OK to just dumb-join the scheme with the path.
	aliasURL := m.url.Scheme + "://" + m.opath
	path := sourceCachePath(cachedir, aliasURL)
	ustr := m.url.String()

	r, err := vcs.NewGitRepo(ustr, path)
	if err != nil {
		os.RemoveAll(path)
		r, err = vcs.NewGitRepo(ustr, path)
		if err != nil {
			return nil, unwrapVcsErr(err)
		}
	}

	return &gopkginSource{
		gitSource: gitSource{
			baseVCSSource: baseVCSSource{
				repo: &gitRepo{r},
			},
		},
		major:    m.major,
		unstable: m.unstable,
		aliasURL: aliasURL,
	}, nil
}

func (m maybeGopkginSource) URL() *url.URL {
	return &url.URL{
		Scheme: m.url.Scheme,
		Path:   m.opath,
	}
}

func (m maybeGopkginSource) String() string {
	return fmt.Sprintf("%T: %s (v%v) %s ", m, m.opath, m.major, ufmt(m.url))
}

type maybeBzrSource struct {
	url *url.URL
}

func (m maybeBzrSource) try(ctx context.Context, cachedir string) (source, error) {
	ustr := m.url.String()
	path := sourceCachePath(cachedir, ustr)

	r, err := vcs.NewBzrRepo(ustr, path)
	if err != nil {
		os.RemoveAll(path)
		r, err = vcs.NewBzrRepo(ustr, path)
		if err != nil {
			return nil, unwrapVcsErr(err)
		}
	}

	return &bzrSource{
		baseVCSSource: baseVCSSource{
			repo: &bzrRepo{r},
		},
	}, nil
}

func (m maybeBzrSource) URL() *url.URL {
	return m.url
}

func (m maybeBzrSource) String() string {
	return fmt.Sprintf("%T: %s", m, ufmt(m.url))
}

type maybeHgSource struct {
	url *url.URL
}

func (m maybeHgSource) try(ctx context.Context, cachedir string) (source, error) {
	ustr := m.url.String()
	path := sourceCachePath(cachedir, ustr)

	r, err := vcs.NewHgRepo(ustr, path)
	if err != nil {
		os.RemoveAll(path)
		r, err = vcs.NewHgRepo(ustr, path)
		if err != nil {
			return nil, unwrapVcsErr(err)
		}
	}

	return &hgSource{
		baseVCSSource: baseVCSSource{
			repo: &hgRepo{r},
		},
	}, nil
}

func (m maybeHgSource) URL() *url.URL {
	return m.url
}

func (m maybeHgSource) String() string {
	return fmt.Sprintf("%T: %s", m, ufmt(m.url))
}

// borrow from stdlib
// more useful string for debugging than fmt's struct printer
func ufmt(u *url.URL) string {
	var user, pass interface{}
	if u.User != nil {
		user = u.User.Username()
		if p, ok := u.User.Password(); ok {
			pass = p
		}
	}
	return fmt.Sprintf("host=%q, path=%q, opaque=%q, scheme=%q, user=%#v, pass=%#v, rawpath=%q, rawq=%q, frag=%q",
		u.Host, u.Path, u.Opaque, u.Scheme, user, pass, u.RawPath, u.RawQuery, u.Fragment)
}
