// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gps

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/golang/dep/gps/pkgtree"
)

// sourceCache is an interface for creating singleSourceCaches, and safely
// releasing backing resources via close.
type sourceCache interface {
	// newSingleSourceCache creates a new singleSourceCache for id, which
	// remains valid until close is called.
	newSingleSourceCache(id ProjectIdentifier) singleSourceCache
	// close releases background resources.
	close() error
}

// singleSourceCache provides a method set for storing and retrieving data about
// a single source.
type singleSourceCache interface {
	// Store the manifest and lock information for a given revision, as defined by
	// a particular ProjectAnalyzer.
	setManifestAndLock(Revision, ProjectAnalyzerInfo, Manifest, Lock)

	// Get the manifest and lock information for a given revision, as defined by
	// a particular ProjectAnalyzer.
	getManifestAndLock(Revision, ProjectAnalyzerInfo) (Manifest, Lock, bool)

	// Store a PackageTree for a given revision.
	setPackageTree(Revision, pkgtree.PackageTree)

	// Get the PackageTree for a given revision.
	getPackageTree(Revision, ProjectRoot) (pkgtree.PackageTree, bool)

	// Indicate to the cache that an individual revision is known to exist.
	markRevisionExists(r Revision)

	// Store the mappings between a set of PairedVersions' surface versions
	// their corresponding revisions.
	//
	// The existing list of versions will be purged before writing. Revisions
	// will have their pairings purged, but record of the revision existing will
	// be kept, on the assumption that revisions are immutable and permanent.
	setVersionMap(versionList []PairedVersion)

	// Get the list of unpaired versions corresponding to the given revision.
	getVersionsFor(Revision) ([]UnpairedVersion, bool)

	// Gets all the version pairs currently known to the cache.
	getAllVersions() ([]PairedVersion, bool)

	// Get the revision corresponding to the given unpaired version.
	getRevisionFor(UnpairedVersion) (Revision, bool)

	// Attempt to convert the given Version to a Revision, given information
	// currently present in the cache, and in the Version itself.
	toRevision(v Version) (Revision, bool)

	// Attempt to convert the given Version to an UnpairedVersion, given
	// information currently present in the cache, or in the Version itself.
	//
	// If the input is a revision and multiple UnpairedVersions are associated
	// with it, whatever happens to be the first is returned.
	toUnpaired(v Version) (UnpairedVersion, bool)
}

// memoryCache is a sourceCache which creates singleSourceCacheMemory instances.
type memoryCache struct{}

func (memoryCache) newSingleSourceCache(ProjectIdentifier) singleSourceCache {
	return newMemoryCache()
}

func (memoryCache) close() error { return nil }

type singleSourceCacheMemory struct {
	// Protects all fields.
	mut   sync.RWMutex
	infos map[ProjectAnalyzerInfo]map[Revision]projectInfo
	// Replaced, never modified. Imports are *relative* (ImportRoot prefix trimmed).
	ptrees map[Revision]map[string]pkgtree.PackageOrErr
	// Replaced, never modified.
	vList []PairedVersion
	vMap  map[UnpairedVersion]Revision
	rMap  map[Revision][]UnpairedVersion
}

func newMemoryCache() singleSourceCache {
	return &singleSourceCacheMemory{
		infos:  make(map[ProjectAnalyzerInfo]map[Revision]projectInfo),
		ptrees: make(map[Revision]map[string]pkgtree.PackageOrErr),
		vMap:   make(map[UnpairedVersion]Revision),
		rMap:   make(map[Revision][]UnpairedVersion),
	}
}

type projectInfo struct {
	Manifest
	Lock
}

func (c *singleSourceCacheMemory) setManifestAndLock(r Revision, pai ProjectAnalyzerInfo, m Manifest, l Lock) {
	c.mut.Lock()
	inner, has := c.infos[pai]
	if !has {
		inner = make(map[Revision]projectInfo)
		c.infos[pai] = inner
	}
	inner[r] = projectInfo{Manifest: m, Lock: l}

	// Ensure there's at least an entry in the rMap so that the rMap always has
	// a complete picture of the revisions we know to exist
	if _, has = c.rMap[r]; !has {
		c.rMap[r] = nil
	}
	c.mut.Unlock()
}

func (c *singleSourceCacheMemory) getManifestAndLock(r Revision, pai ProjectAnalyzerInfo) (Manifest, Lock, bool) {
	c.mut.Lock()
	defer c.mut.Unlock()

	inner, has := c.infos[pai]
	if !has {
		return nil, nil, false
	}

	pi, has := inner[r]
	if has {
		return pi.Manifest, pi.Lock, true
	}
	return nil, nil, false
}

func (c *singleSourceCacheMemory) setPackageTree(r Revision, ptree pkgtree.PackageTree) {
	// Make a copy, with relative import paths.
	pkgs := pkgtree.CopyPackages(ptree.Packages, func(ip string, poe pkgtree.PackageOrErr) (string, pkgtree.PackageOrErr) {
		poe.P.ImportPath = "" // Don't store this
		return strings.TrimPrefix(ip, ptree.ImportRoot), poe
	})

	c.mut.Lock()
	c.ptrees[r] = pkgs

	// Ensure there's at least an entry in the rMap so that the rMap always has
	// a complete picture of the revisions we know to exist
	if _, has := c.rMap[r]; !has {
		c.rMap[r] = nil
	}
	c.mut.Unlock()
}

func (c *singleSourceCacheMemory) getPackageTree(r Revision, pr ProjectRoot) (pkgtree.PackageTree, bool) {
	c.mut.Lock()
	rptree, has := c.ptrees[r]
	c.mut.Unlock()

	if !has {
		return pkgtree.PackageTree{}, false
	}

	// Return a copy, with full import paths.
	pkgs := pkgtree.CopyPackages(rptree, func(rpath string, poe pkgtree.PackageOrErr) (string, pkgtree.PackageOrErr) {
		ip := path.Join(string(pr), rpath)
		if poe.Err == nil {
			poe.P.ImportPath = ip
		}
		return ip, poe
	})

	return pkgtree.PackageTree{
		ImportRoot: string(pr),
		Packages:   pkgs,
	}, true
}

func (c *singleSourceCacheMemory) setVersionMap(versionList []PairedVersion) {
	c.mut.Lock()
	c.vList = versionList
	// TODO(sdboyer) how do we handle cache consistency here - revs that may
	// be out of date vis-a-vis the ptrees or infos maps?
	for r := range c.rMap {
		c.rMap[r] = nil
	}

	c.vMap = make(map[UnpairedVersion]Revision, len(versionList))

	for _, pv := range versionList {
		u, r := pv.Unpair(), pv.Revision()
		c.vMap[u] = r
		c.rMap[r] = append(c.rMap[r], u)
	}
	c.mut.Unlock()
}

func (c *singleSourceCacheMemory) markRevisionExists(r Revision) {
	c.mut.Lock()
	if _, has := c.rMap[r]; !has {
		c.rMap[r] = nil
	}
	c.mut.Unlock()
}

func (c *singleSourceCacheMemory) getVersionsFor(r Revision) ([]UnpairedVersion, bool) {
	c.mut.Lock()
	versionList, has := c.rMap[r]
	c.mut.Unlock()
	return versionList, has
}

func (c *singleSourceCacheMemory) getAllVersions() ([]PairedVersion, bool) {
	c.mut.Lock()
	vList := c.vList
	c.mut.Unlock()

	if vList == nil {
		return nil, false
	}
	cp := make([]PairedVersion, len(vList))
	copy(cp, vList)
	return cp, true
}

func (c *singleSourceCacheMemory) getRevisionFor(uv UnpairedVersion) (Revision, bool) {
	c.mut.Lock()
	r, has := c.vMap[uv]
	c.mut.Unlock()
	return r, has
}

func (c *singleSourceCacheMemory) toRevision(v Version) (Revision, bool) {
	switch t := v.(type) {
	case Revision:
		return t, true
	case PairedVersion:
		return t.Revision(), true
	case UnpairedVersion:
		c.mut.Lock()
		r, has := c.vMap[t]
		c.mut.Unlock()
		return r, has
	default:
		panic(fmt.Sprintf("Unknown version type %T", v))
	}
}

func (c *singleSourceCacheMemory) toUnpaired(v Version) (UnpairedVersion, bool) {
	switch t := v.(type) {
	case UnpairedVersion:
		return t, true
	case PairedVersion:
		return t.Unpair(), true
	case Revision:
		c.mut.Lock()
		upv, has := c.rMap[t]
		c.mut.Unlock()

		if has && len(upv) > 0 {
			return upv[0], true
		}
		return nil, false
	default:
		panic(fmt.Sprintf("unknown version type %T", v))
	}
}

// TODO(sdboyer) remove once source caching can be moved into separate package
func locksAreEq(l1, l2 Lock) bool {
	ii1, ii2 := l1.InputImports(), l2.InputImports()
	if len(ii1) != len(ii2) {
		return false
	}

	ilen := len(ii1)
	if ilen > 0 {
		sort.Strings(ii1)
		sort.Strings(ii2)
		for i := 0; i < ilen; i++ {
			if ii1[i] != ii2[i] {
				return false
			}
		}
	}

	p1, p2 := l1.Projects(), l2.Projects()
	if len(p1) != len(p2) {
		return false
	}

	p1, p2 = sortLockedProjects(p1), sortLockedProjects(p2)

	for k, lp := range p1 {
		if !lp.Eq(p2[k]) {
			return false
		}
	}
	return true
}
