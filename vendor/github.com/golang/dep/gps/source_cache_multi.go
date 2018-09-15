// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gps

import (
	"github.com/golang/dep/gps/pkgtree"
)

// multiCache creates singleSourceMultiCaches, and coordinates their async updates.
type multiCache struct {
	mem, disk sourceCache
	// Asynchronous disk cache updates. Closed by the close method.
	async chan func()
	// Closed when async has completed processing.
	done chan struct{}
}

// newMultiCache returns a new multiCache backed by mem and disk sourceCaches.
// Spawns a single background goroutine which lives until close() is called.
func newMultiCache(mem, disk sourceCache) *multiCache {
	m := &multiCache{
		mem:   mem,
		disk:  disk,
		async: make(chan func(), 50),
		done:  make(chan struct{}),
	}
	go m.processAsync()
	return m
}

func (c *multiCache) processAsync() {
	for f := range c.async {
		f()
	}
	close(c.done)
}

// close releases resources after blocking until async writes complete.
func (c *multiCache) close() error {
	close(c.async)
	_ = c.mem.close()
	<-c.done
	return c.disk.close()
}

// newSingleSourceCache returns a singleSourceMultiCache for id.
func (c *multiCache) newSingleSourceCache(id ProjectIdentifier) singleSourceCache {
	return &singleSourceMultiCache{
		mem:   c.mem.newSingleSourceCache(id),
		disk:  c.disk.newSingleSourceCache(id),
		async: c.async,
	}
}

// singleSourceMultiCache manages two cache levels, ephemeral in-memory and persistent on-disk.
//
// The in-memory cache is always checked first, with the on-disk used as a fallback.
// Values read from disk are set in-memory when an appropriate method exists.
//
// Set values are cached both in-memory and on-disk. Values are set synchronously
// in-memory. Writes to the on-disk cache are asynchronous, and executed in order by a
// background goroutine.
type singleSourceMultiCache struct {
	mem, disk singleSourceCache
	// Asynchronous disk cache updates.
	async chan<- func()
}

func (c *singleSourceMultiCache) setManifestAndLock(r Revision, ai ProjectAnalyzerInfo, m Manifest, l Lock) {
	c.mem.setManifestAndLock(r, ai, m, l)
	c.async <- func() { c.disk.setManifestAndLock(r, ai, m, l) }
}

func (c *singleSourceMultiCache) getManifestAndLock(r Revision, ai ProjectAnalyzerInfo) (Manifest, Lock, bool) {
	m, l, ok := c.mem.getManifestAndLock(r, ai)
	if ok {
		return m, l, true
	}

	m, l, ok = c.disk.getManifestAndLock(r, ai)
	if ok {
		c.mem.setManifestAndLock(r, ai, m, l)
		return m, l, true
	}

	return nil, nil, false
}

func (c *singleSourceMultiCache) setPackageTree(r Revision, ptree pkgtree.PackageTree) {
	c.mem.setPackageTree(r, ptree)
	c.async <- func() { c.disk.setPackageTree(r, ptree) }
}

func (c *singleSourceMultiCache) getPackageTree(r Revision, pr ProjectRoot) (pkgtree.PackageTree, bool) {
	ptree, ok := c.mem.getPackageTree(r, pr)
	if ok {
		return ptree, true
	}

	ptree, ok = c.disk.getPackageTree(r, pr)
	if ok {
		c.mem.setPackageTree(r, ptree)
		return ptree, true
	}

	return pkgtree.PackageTree{}, false
}

func (c *singleSourceMultiCache) markRevisionExists(r Revision) {
	c.mem.markRevisionExists(r)
	c.async <- func() { c.disk.markRevisionExists(r) }
}

func (c *singleSourceMultiCache) setVersionMap(pvs []PairedVersion) {
	c.mem.setVersionMap(pvs)
	c.async <- func() { c.disk.setVersionMap(pvs) }
}

func (c *singleSourceMultiCache) getVersionsFor(rev Revision) ([]UnpairedVersion, bool) {
	uvs, ok := c.mem.getVersionsFor(rev)
	if ok {
		return uvs, true
	}

	return c.disk.getVersionsFor(rev)
}

func (c *singleSourceMultiCache) getAllVersions() ([]PairedVersion, bool) {
	pvs, ok := c.mem.getAllVersions()
	if ok {
		return pvs, true
	}

	pvs, ok = c.disk.getAllVersions()
	if ok {
		c.mem.setVersionMap(pvs)
		return pvs, true
	}

	return nil, false
}

func (c *singleSourceMultiCache) getRevisionFor(uv UnpairedVersion) (Revision, bool) {
	rev, ok := c.mem.getRevisionFor(uv)
	if ok {
		return rev, true
	}

	return c.disk.getRevisionFor(uv)
}

func (c *singleSourceMultiCache) toRevision(v Version) (Revision, bool) {
	rev, ok := c.mem.toRevision(v)
	if ok {
		return rev, true
	}

	return c.disk.toRevision(v)
}

func (c *singleSourceMultiCache) toUnpaired(v Version) (UnpairedVersion, bool) {
	uv, ok := c.mem.toUnpaired(v)
	if ok {
		return uv, true
	}

	return c.disk.toUnpaired(v)
}
