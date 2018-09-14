// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gps

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/golang/dep/gps/pkgtree"
	"github.com/pkg/errors"
)

// sourceState represent the states that a source can be in, depending on how
// much search and discovery work ahs been done by a source's managing gateway.
//
// These are basically used to achieve a cheap approximation of a FSM.
type sourceState int32

const (
	// sourceExistsUpstream means the chosen source was verified upstream, during this execution.
	sourceExistsUpstream sourceState = 1 << iota
	// sourceExistsLocally means the repo was retrieved in the past.
	sourceExistsLocally
	// sourceHasLatestVersionList means the version list was refreshed within the cache window.
	sourceHasLatestVersionList
	// sourceHasLatestLocally means the repo was pulled fresh during this execution.
	sourceHasLatestLocally
)

func (state sourceState) String() string {
	var b bytes.Buffer
	for _, s := range []struct {
		sourceState
		string
	}{
		{sourceExistsUpstream, "sourceExistsUpstream"},
		{sourceExistsLocally, "sourceExistsLocally"},
		{sourceHasLatestVersionList, "sourceHasLatestVersionList"},
		{sourceHasLatestLocally, "sourceHasLatestLocally"},
	} {
		if state&s.sourceState > 0 {
			if b.Len() > 0 {
				b.WriteString("|")
			}
			b.WriteString(s.string)
		}
	}
	return b.String()
}

type srcReturn struct {
	*sourceGateway
	error
}

type sourceCoordinator struct {
	supervisor *supervisor
	deducer    deducer
	srcmut     sync.RWMutex // guards srcs and srcIdx
	srcs       map[string]*sourceGateway
	nameToURL  map[string]string
	psrcmut    sync.Mutex // guards protoSrcs map
	protoSrcs  map[string][]chan srcReturn
	cachedir   string
	cache      sourceCache
	logger     *log.Logger
}

// newSourceCoordinator returns a new sourceCoordinator.
// Passing a nil sourceCache defaults to an in-memory cache.
func newSourceCoordinator(superv *supervisor, deducer deducer, cachedir string, cache sourceCache, logger *log.Logger) *sourceCoordinator {
	if cache == nil {
		cache = memoryCache{}
	}
	return &sourceCoordinator{
		supervisor: superv,
		deducer:    deducer,
		cachedir:   cachedir,
		cache:      cache,
		logger:     logger,
		srcs:       make(map[string]*sourceGateway),
		nameToURL:  make(map[string]string),
		protoSrcs:  make(map[string][]chan srcReturn),
	}
}

func (sc *sourceCoordinator) close() {
	if err := sc.cache.close(); err != nil {
		sc.logger.Println(errors.Wrap(err, "failed to close the source cache"))
	}
}

func (sc *sourceCoordinator) getSourceGatewayFor(ctx context.Context, id ProjectIdentifier) (*sourceGateway, error) {
	if err := sc.supervisor.ctx.Err(); err != nil {
		return nil, err
	}

	normalizedName := id.normalizedSource()

	sc.srcmut.RLock()
	if url, has := sc.nameToURL[normalizedName]; has {
		srcGate, has := sc.srcs[url]
		sc.srcmut.RUnlock()
		if has {
			return srcGate, nil
		}
		panic(fmt.Sprintf("%q was URL for %q in nameToURL, but no corresponding srcGate in srcs map", url, normalizedName))
	}

	// Without a direct match, we must fold the input name to a generally
	// stable, caseless variant and primarily work from that. This ensures that
	// on case-insensitive filesystems, we do not end up with multiple
	// sourceGateways for paths that vary only by case. We perform folding
	// unconditionally, independent of whether the underlying fs is
	// case-sensitive, in order to ensure uniform behavior.
	//
	// This has significant implications. It is effectively deciding that the
	// ProjectRoot portion of import paths are case-insensitive, which is by no
	// means an invariant maintained by all hosting systems. If this presents a
	// problem in practice, then we can explore expanding the deduction system
	// to include case-sensitivity-for-roots metadata and treat it on a
	// host-by-host basis. Such cases would still be rejected by the Go
	// toolchain's compiler, though, and case-sensitivity in root names is
	// likely to be at least frowned on if not disallowed by most hosting
	// systems. So we follow this path, which is both a vastly simpler solution
	// and one that seems quite likely to work in practice.
	foldedNormalName := toFold(normalizedName)
	notFolded := foldedNormalName != normalizedName
	if notFolded {
		// If the folded name differs from the input name, then there may
		// already be an entry for it in the nameToURL map, so check again.
		if url, has := sc.nameToURL[foldedNormalName]; has {
			srcGate, has := sc.srcs[url]
			// There was a match on the canonical folded variant. Upgrade to a
			// write lock, so that future calls on this name don't need to
			// burn cycles on folding.
			sc.srcmut.RUnlock()
			sc.srcmut.Lock()
			// It may be possible that another goroutine could interleave
			// between the unlock and re-lock. Even if they do, though, they'll
			// only have recorded the same url value as we have here. In other
			// words, these operations commute, so we can safely write here
			// without checking again.
			sc.nameToURL[normalizedName] = url
			sc.srcmut.Unlock()
			if has {
				return srcGate, nil
			}
			panic(fmt.Sprintf("%q was URL for %q in nameToURL, but no corresponding srcGate in srcs map", url, normalizedName))
		}
	}
	sc.srcmut.RUnlock()

	// No gateway exists for this path yet; set up a proto, being careful to fold
	// together simultaneous attempts on the same case-folded path.
	sc.psrcmut.Lock()
	if chans, has := sc.protoSrcs[foldedNormalName]; has {
		// Another goroutine is already working on this normalizedName. Fold
		// in with that work by attaching our return channels to the list.
		rc := make(chan srcReturn, 1)
		sc.protoSrcs[foldedNormalName] = append(chans, rc)
		sc.psrcmut.Unlock()
		ret := <-rc
		return ret.sourceGateway, ret.error
	}

	sc.protoSrcs[foldedNormalName] = []chan srcReturn{}
	sc.psrcmut.Unlock()

	doReturn := func(sg *sourceGateway, err error) {
		ret := srcReturn{sourceGateway: sg, error: err}
		sc.psrcmut.Lock()
		for _, rc := range sc.protoSrcs[foldedNormalName] {
			rc <- ret
		}
		delete(sc.protoSrcs, foldedNormalName)
		sc.psrcmut.Unlock()
	}

	pd, err := sc.deducer.deduceRootPath(ctx, normalizedName)
	if err != nil {
		// As in the deducer, don't cache errors so that externally-driven retry
		// strategies can be constructed.
		doReturn(nil, err)
		return nil, err
	}

	// It'd be quite the feat - but not impossible - for a gateway
	// corresponding to this normalizedName to have slid into the main
	// sources map after the initial unlock, but before this goroutine got
	// scheduled. Guard against that by checking the main sources map again
	// and bailing out if we find an entry.
	sc.srcmut.RLock()
	if url, has := sc.nameToURL[foldedNormalName]; has {
		if srcGate, has := sc.srcs[url]; has {
			sc.srcmut.RUnlock()
			doReturn(srcGate, nil)
			return srcGate, nil
		}
		panic(fmt.Sprintf("%q was URL for %q in nameToURL, but no corresponding srcGate in srcs map", url, normalizedName))
	}
	sc.srcmut.RUnlock()

	sc.srcmut.Lock()
	defer sc.srcmut.Unlock()

	// Get or create a sourceGateway.
	var srcGate *sourceGateway
	var url, unfoldedURL string
	var errs errorSlice
	for _, m := range pd.mb {
		url = m.URL().String()
		if notFolded {
			// If the normalizedName and foldedNormalName differ, then we're pretty well
			// guaranteed that returned URL will also need folding into canonical form.
			unfoldedURL = url
			url = toFold(url)
		}
		if sg, has := sc.srcs[url]; has {
			srcGate = sg
			break
		}
		src, err := m.try(ctx, sc.cachedir)
		if err == nil {
			cache := sc.cache.newSingleSourceCache(id)
			srcGate, err = newSourceGateway(ctx, src, sc.supervisor, sc.cachedir, cache)
			if err == nil {
				sc.srcs[url] = srcGate
				break
			}
		}
		errs = append(errs, err)
	}
	if srcGate == nil {
		doReturn(nil, errs)
		return nil, errs
	}

	// Record the name -> URL mapping, making sure that we also get the
	// self-mapping.
	sc.nameToURL[foldedNormalName] = url
	if url != foldedNormalName {
		sc.nameToURL[url] = url
	}

	// Make sure we have both the folded and unfolded names and URLs recorded in
	// the map, if the input needed folding.
	if notFolded {
		sc.nameToURL[normalizedName] = url
		sc.nameToURL[unfoldedURL] = url
	}

	doReturn(srcGate, nil)
	return srcGate, nil
}

// sourceGateways manage all incoming calls for data from sources, serializing
// and caching them as needed.
type sourceGateway struct {
	cachedir string
	srcState sourceState
	src      source
	cache    singleSourceCache
	mu       sync.Mutex // global lock, serializes all behaviors
	suprvsr  *supervisor
}

// newSourceGateway returns a new gateway for src. If the source exists locally,
// the local state may be cleaned, otherwise we ping upstream.
func newSourceGateway(ctx context.Context, src source, superv *supervisor, cachedir string, cache singleSourceCache) (*sourceGateway, error) {
	var state sourceState
	local := src.existsLocally(ctx)
	if local {
		state |= sourceExistsLocally
		if err := superv.do(ctx, src.upstreamURL(), ctValidateLocal, func(ctx context.Context) error {
			return src.maybeClean(ctx)
		}); err != nil {
			return nil, err
		}
	}

	sg := &sourceGateway{
		srcState: state,
		src:      src,
		cachedir: cachedir,
		cache:    cache,
		suprvsr:  superv,
	}

	if !local {
		if err := sg.require(ctx, sourceExistsUpstream); err != nil {
			return nil, err
		}
	}

	return sg, nil
}

func (sg *sourceGateway) syncLocal(ctx context.Context) error {
	sg.mu.Lock()
	err := sg.require(ctx, sourceExistsLocally|sourceHasLatestLocally)
	sg.mu.Unlock()
	return err
}

func (sg *sourceGateway) existsInCache(ctx context.Context) error {
	sg.mu.Lock()
	err := sg.require(ctx, sourceExistsLocally)
	sg.mu.Unlock()
	return err
}

func (sg *sourceGateway) existsUpstream(ctx context.Context) error {
	sg.mu.Lock()
	err := sg.require(ctx, sourceExistsUpstream)
	sg.mu.Unlock()
	return err
}

func (sg *sourceGateway) exportVersionTo(ctx context.Context, v Version, to string) error {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	err := sg.require(ctx, sourceExistsLocally)
	if err != nil {
		return err
	}

	r, err := sg.convertToRevision(ctx, v)
	if err != nil {
		return err
	}

	err = sg.suprvsr.do(ctx, sg.src.upstreamURL(), ctExportTree, func(ctx context.Context) error {
		return sg.src.exportRevisionTo(ctx, r, to)
	})

	// It's possible (in git) that we may have tried this against a version that
	// doesn't exist in the repository cache, even though we know it exists in
	// the upstream. If it looks like that might be the case, update the local
	// and retry.
	// TODO(sdboyer) It'd be better if we could check the error to see if this
	// actually was the cause of the problem.
	if err != nil && sg.srcState&sourceHasLatestLocally == 0 {
		if err = sg.require(ctx, sourceHasLatestLocally); err == nil {
			err = sg.suprvsr.do(ctx, sg.src.upstreamURL(), ctExportTree, func(ctx context.Context) error {
				return sg.src.exportRevisionTo(ctx, r, to)
			})
		}
	}

	return err
}

func (sg *sourceGateway) exportPrunedVersionTo(ctx context.Context, lp LockedProject, prune PruneOptions, to string) error {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	err := sg.require(ctx, sourceExistsLocally)
	if err != nil {
		return err
	}

	r, err := sg.convertToRevision(ctx, lp.Version())
	if err != nil {
		return err
	}

	if fastprune, ok := sg.src.(sourceFastPrune); ok {
		return sg.suprvsr.do(ctx, sg.src.upstreamURL(), ctExportTree, func(ctx context.Context) error {
			return fastprune.exportPrunedRevisionTo(ctx, r, lp.Packages(), prune, to)
		})
	}

	if err = sg.suprvsr.do(ctx, sg.src.upstreamURL(), ctExportTree, func(ctx context.Context) error {
		return sg.src.exportRevisionTo(ctx, r, to)
	}); err != nil {
		return err
	}

	return PruneProject(to, lp, prune)
}

func (sg *sourceGateway) getManifestAndLock(ctx context.Context, pr ProjectRoot, v Version, an ProjectAnalyzer) (Manifest, Lock, error) {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	r, err := sg.convertToRevision(ctx, v)
	if err != nil {
		return nil, nil, err
	}

	m, l, has := sg.cache.getManifestAndLock(r, an.Info())
	if has {
		return m, l, nil
	}

	err = sg.require(ctx, sourceExistsLocally)
	if err != nil {
		return nil, nil, err
	}

	label := fmt.Sprintf("%s:%s", sg.src.upstreamURL(), an.Info())
	err = sg.suprvsr.do(ctx, label, ctGetManifestAndLock, func(ctx context.Context) error {
		m, l, err = sg.src.getManifestAndLock(ctx, pr, r, an)
		return err
	})

	// It's possible (in git) that we may have tried this against a version that
	// doesn't exist in the repository cache, even though we know it exists in
	// the upstream. If it looks like that might be the case, update the local
	// and retry.
	// TODO(sdboyer) It'd be better if we could check the error to see if this
	// actually was the cause of the problem.
	if err != nil && sg.srcState&sourceHasLatestLocally == 0 {
		// TODO(sdboyer) we should warn/log/something in adaptive recovery
		// situations like this
		err = sg.require(ctx, sourceHasLatestLocally)
		if err != nil {
			return nil, nil, err
		}

		err = sg.suprvsr.do(ctx, label, ctGetManifestAndLock, func(ctx context.Context) error {
			m, l, err = sg.src.getManifestAndLock(ctx, pr, r, an)
			return err
		})
	}

	if err != nil {
		return nil, nil, err
	}

	sg.cache.setManifestAndLock(r, an.Info(), m, l)
	return m, l, nil
}

func (sg *sourceGateway) listPackages(ctx context.Context, pr ProjectRoot, v Version) (pkgtree.PackageTree, error) {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	r, err := sg.convertToRevision(ctx, v)
	if err != nil {
		return pkgtree.PackageTree{}, err
	}

	ptree, has := sg.cache.getPackageTree(r, pr)
	if has {
		return ptree, nil
	}

	err = sg.require(ctx, sourceExistsLocally)
	if err != nil {
		return pkgtree.PackageTree{}, err
	}

	label := fmt.Sprintf("%s:%s", pr, sg.src.upstreamURL())
	err = sg.suprvsr.do(ctx, label, ctListPackages, func(ctx context.Context) error {
		ptree, err = sg.src.listPackages(ctx, pr, r)
		return err
	})

	// It's possible (in git) that we may have tried this against a version that
	// doesn't exist in the repository cache, even though we know it exists in
	// the upstream. If it looks like that might be the case, update the local
	// and retry.
	// TODO(sdboyer) It'd be better if we could check the error to see if this
	// actually was the cause of the problem.
	if err != nil && sg.srcState&sourceHasLatestLocally == 0 {
		// TODO(sdboyer) we should warn/log/something in adaptive recovery
		// situations like this
		err = sg.require(ctx, sourceHasLatestLocally)
		if err != nil {
			return pkgtree.PackageTree{}, err
		}

		err = sg.suprvsr.do(ctx, label, ctListPackages, func(ctx context.Context) error {
			ptree, err = sg.src.listPackages(ctx, pr, r)
			return err
		})
	}

	if err != nil {
		return pkgtree.PackageTree{}, err
	}

	sg.cache.setPackageTree(r, ptree)
	return ptree, nil
}

// caller must hold sg.mu.
func (sg *sourceGateway) convertToRevision(ctx context.Context, v Version) (Revision, error) {
	// When looking up by Version, there are four states that may have
	// differing opinions about version->revision mappings:
	//
	//   1. The upstream source/repo (canonical)
	//   2. The local source/repo
	//   3. The local cache
	//   4. The input (params to this method)
	//
	// If the input differs from any of the above, it's likely because some lock
	// got written somewhere with a version/rev pair that has since changed or
	// been removed. But correct operation dictates that such a mis-mapping be
	// respected; if the mis-mapping is to be corrected, it has to be done
	// intentionally by the caller, not automatically here.
	r, has := sg.cache.toRevision(v)
	if has {
		return r, nil
	}

	if sg.srcState&sourceHasLatestVersionList != 0 {
		// We have the latest version list already and didn't get a match, so
		// this is definitely a failure case.
		return "", fmt.Errorf("version %q does not exist in source", v)
	}

	// The version list is out of date; it's possible this version might
	// show up after loading it.
	err := sg.require(ctx, sourceHasLatestVersionList)
	if err != nil {
		return "", err
	}

	r, has = sg.cache.toRevision(v)
	if !has {
		return "", fmt.Errorf("version %q does not exist in source", v)
	}

	return r, nil
}

func (sg *sourceGateway) listVersions(ctx context.Context) ([]PairedVersion, error) {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	if pvs, ok := sg.cache.getAllVersions(); ok {
		return pvs, nil
	}

	err := sg.require(ctx, sourceHasLatestVersionList)
	if err != nil {
		return nil, err
	}
	if pvs, ok := sg.cache.getAllVersions(); ok {
		return pvs, nil
	}
	return nil, nil
}

func (sg *sourceGateway) revisionPresentIn(ctx context.Context, r Revision) (bool, error) {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	err := sg.require(ctx, sourceExistsLocally)
	if err != nil {
		return false, err
	}

	if _, exists := sg.cache.getVersionsFor(r); exists {
		return true, nil
	}

	present, err := sg.src.revisionPresentIn(r)
	if err == nil && present {
		sg.cache.markRevisionExists(r)
	}
	return present, err
}

func (sg *sourceGateway) disambiguateRevision(ctx context.Context, r Revision) (Revision, error) {
	sg.mu.Lock()
	defer sg.mu.Unlock()

	err := sg.require(ctx, sourceExistsLocally)
	if err != nil {
		return "", err
	}

	return sg.src.disambiguateRevision(ctx, r)
}

// sourceExistsUpstream verifies that the source exists upstream and that the
// upstreamURL has not changed and returns any additional sourceState, or an error.
func (sg *sourceGateway) sourceExistsUpstream(ctx context.Context) (sourceState, error) {
	if sg.src.existsCallsListVersions() {
		return sg.loadLatestVersionList(ctx)
	}
	err := sg.suprvsr.do(ctx, sg.src.sourceType(), ctSourcePing, func(ctx context.Context) error {
		if !sg.src.existsUpstream(ctx) {
			return errors.Errorf("source does not exist upstream: %s: %s", sg.src.sourceType(), sg.src.upstreamURL())
		}
		return nil
	})
	return 0, err
}

// initLocal initializes the source locally and returns the resulting sourceState.
func (sg *sourceGateway) initLocal(ctx context.Context) (sourceState, error) {
	if err := sg.suprvsr.do(ctx, sg.src.sourceType(), ctSourceInit, func(ctx context.Context) error {
		err := sg.src.initLocal(ctx)
		return errors.Wrapf(err, "failed to fetch source for %s", sg.src.upstreamURL())
	}); err != nil {
		return 0, err
	}
	return sourceExistsUpstream | sourceExistsLocally | sourceHasLatestLocally, nil
}

// loadLatestVersionList loads the latest version list, possibly ensuring the source
// exists locally first, and returns the resulting sourceState.
func (sg *sourceGateway) loadLatestVersionList(ctx context.Context) (sourceState, error) {
	var addlState sourceState
	if sg.src.listVersionsRequiresLocal() && !sg.src.existsLocally(ctx) {
		as, err := sg.initLocal(ctx)
		if err != nil {
			return 0, err
		}
		addlState |= as
	}
	var pvl []PairedVersion
	if err := sg.suprvsr.do(ctx, sg.src.sourceType(), ctListVersions, func(ctx context.Context) error {
		var err error
		pvl, err = sg.src.listVersions(ctx)
		return errors.Wrapf(err, "failed to list versions for %s", sg.src.upstreamURL())
	}); err != nil {
		return addlState, err
	}
	sg.cache.setVersionMap(pvl)
	return addlState | sourceHasLatestVersionList, nil
}

// require ensures the sourceGateway has the wanted sourceState, fetching more
// data if necessary. Returns an error if the state could not be reached.
// caller must hold sg.mu
func (sg *sourceGateway) require(ctx context.Context, wanted sourceState) (err error) {
	todo := (^sg.srcState) & wanted
	var flag sourceState = 1

	for todo != 0 {
		if todo&flag != 0 {
			// Set up addlState so that individual ops can easily attach
			// more states that were incidentally satisfied by the op.
			var addlState sourceState

			switch flag {
			case sourceExistsUpstream:
				addlState, err = sg.sourceExistsUpstream(ctx)
			case sourceExistsLocally:
				if !sg.src.existsLocally(ctx) {
					addlState, err = sg.initLocal(ctx)
				}
			case sourceHasLatestVersionList:
				if _, ok := sg.cache.getAllVersions(); !ok {
					addlState, err = sg.loadLatestVersionList(ctx)
				}
			case sourceHasLatestLocally:
				err = sg.suprvsr.do(ctx, sg.src.sourceType(), ctSourceFetch, func(ctx context.Context) error {
					return sg.src.updateLocal(ctx)
				})
				addlState = sourceExistsUpstream | sourceExistsLocally
			}

			if err != nil {
				return
			}

			checked := flag | addlState
			sg.srcState |= checked
			todo &= ^checked
		}

		flag <<= 1
	}

	return nil
}

// source is an abstraction around the different underlying types (git, bzr, hg,
// svn, maybe raw on-disk code, and maybe eventually a registry) that can
// provide versioned project source trees.
type source interface {
	existsLocally(context.Context) bool
	existsUpstream(context.Context) bool
	upstreamURL() string
	initLocal(context.Context) error
	updateLocal(context.Context) error
	// maybeClean is a no-op when the underlying source does not support cleaning.
	maybeClean(context.Context) error
	listVersions(context.Context) ([]PairedVersion, error)
	getManifestAndLock(context.Context, ProjectRoot, Revision, ProjectAnalyzer) (Manifest, Lock, error)
	listPackages(context.Context, ProjectRoot, Revision) (pkgtree.PackageTree, error)
	revisionPresentIn(Revision) (bool, error)
	disambiguateRevision(context.Context, Revision) (Revision, error)
	exportRevisionTo(context.Context, Revision, string) error
	sourceType() string
	// existsCallsListVersions returns true if calling existsUpstream actually lists
	// versions underneath, meaning listVersions might as well be used instead.
	existsCallsListVersions() bool
	// listVersionsRequiresLocal returns true if calling listVersions first
	// requires the source to exist locally.
	listVersionsRequiresLocal() bool
}

type sourceFastPrune interface {
	source
	exportPrunedRevisionTo(context.Context, Revision, []string, PruneOptions, string) error
}
