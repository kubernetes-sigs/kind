// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
)

type graphviz struct {
	ps []*gvnode
	b  bytes.Buffer
	h  map[string]uint32
	// clusters is a map of project name and subgraph object. This can be used
	// to refer the subgraph by project name.
	clusters map[string]*gvsubgraph
}

type gvnode struct {
	project  string
	version  string
	children []string
}

// Sort gvnode(s).
type byGvnode []gvnode

func (n byGvnode) Len() int           { return len(n) }
func (n byGvnode) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n byGvnode) Less(i, j int) bool { return n[i].project < n[j].project }

func (g graphviz) New() *graphviz {
	ga := &graphviz{
		ps:       []*gvnode{},
		h:        make(map[string]uint32),
		clusters: make(map[string]*gvsubgraph),
	}
	return ga
}

func (g *graphviz) output(project string) bytes.Buffer {
	if project == "" {
		// Project relations graph.
		g.b.WriteString("digraph {\n\tnode [shape=box];")

		for _, gvp := range g.ps {
			// Create node string
			g.b.WriteString(fmt.Sprintf("\n\t%d [label=\"%s\"];", gvp.hash(), gvp.label()))
		}

		g.createProjectRelations()
	} else {
		// Project-Package relations graph.
		g.b.WriteString("digraph {\n\tnode [shape=box];\n\tcompound=true;\n\tedge [minlen=2];")

		// Declare all the nodes with labels.
		for _, gvp := range g.ps {
			g.b.WriteString(fmt.Sprintf("\n\t%d [label=\"%s\"];", gvp.hash(), gvp.label()))
		}

		// Sort the clusters for a consistent output.
		clusters := sortClusters(g.clusters)

		// Declare all the subgraphs with labels.
		for _, gsg := range clusters {
			g.b.WriteString(fmt.Sprintf("\n\tsubgraph cluster_%d {", gsg.index))
			g.b.WriteString(fmt.Sprintf("\n\t\tlabel = \"%s\";", gsg.project))

			nhashes := []string{}
			for _, pkg := range gsg.packages {
				nhashes = append(nhashes, fmt.Sprint(g.h[pkg]))
			}

			g.b.WriteString(fmt.Sprintf("\n\t\t%s;", strings.Join(nhashes, " ")))
			g.b.WriteString("\n\t}")
		}

		g.createProjectPackageRelations(project, clusters)
	}

	g.b.WriteString("\n}\n")
	return g.b
}

func (g *graphviz) createProjectRelations() {
	// Store relations to avoid duplication
	rels := make(map[string]bool)

	// Create relations
	for _, dp := range g.ps {
		for _, bsc := range dp.children {
			for pr, hsh := range g.h {
				if isPathPrefix(bsc, pr) {
					r := fmt.Sprintf("\n\t%d -> %d", g.h[dp.project], hsh)

					if _, ex := rels[r]; !ex {
						g.b.WriteString(r + ";")
						rels[r] = true
					}

				}
			}
		}
	}
}

func (g *graphviz) createProjectPackageRelations(project string, clusters []*gvsubgraph) {
	// This function takes a child package/project, target project, subgraph meta, from
	// and to of the edge and write a relation.
	linkRelation := func(child, project string, meta []string, from, to uint32) {
		if child == project {
			// Check if it's a cluster.
			target, ok := g.clusters[project]
			if ok {
				// It's a cluster. Point to the Project Root. Use lhead.
				meta = append(meta, fmt.Sprintf("lhead=cluster_%d", target.index))
				// When the head points to a cluster root, use the first
				// node in the cluster as to.
				to = g.h[target.packages[0]]
			}
		}

		if len(meta) > 0 {
			g.b.WriteString(fmt.Sprintf("\n\t%d -> %d [%s];", from, to, strings.Join(meta, " ")))
		} else {
			g.b.WriteString(fmt.Sprintf("\n\t%d -> %d;", from, to))
		}
	}

	// Create relations from nodes.
	for _, node := range g.ps {
		for _, child := range node.children {
			// Only if it points to the target project, proceed further.
			if isPathPrefix(child, project) {
				meta := []string{}
				from := g.h[node.project]
				to := g.h[child]

				linkRelation(child, project, meta, from, to)
			}
		}
	}

	// Create relations from clusters.
	for _, cluster := range clusters {
		for _, child := range cluster.children {
			// Only if it points to the target project, proceed further.
			if isPathPrefix(child, project) {
				meta := []string{fmt.Sprintf("ltail=cluster_%d", cluster.index)}
				// When the tail is from a cluster, use the first node in the
				// cluster as from.
				from := g.h[cluster.packages[0]]
				to := g.h[child]

				linkRelation(child, project, meta, from, to)
			}
		}
	}
}

func (g *graphviz) createNode(project, version string, children []string) {
	pr := &gvnode{
		project:  project,
		version:  version,
		children: children,
	}

	g.h[pr.project] = pr.hash()
	g.ps = append(g.ps, pr)
}

func (dp gvnode) hash() uint32 {
	h := fnv.New32a()
	h.Write([]byte(dp.project))
	return h.Sum32()
}

func (dp gvnode) label() string {
	label := []string{dp.project}

	if dp.version != "" {
		label = append(label, dp.version)
	}

	return strings.Join(label, "\\n")
}

// isPathPrefix ensures that the literal string prefix is a path tree match and
// guards against possibilities like this:
//
// github.com/sdboyer/foo
// github.com/sdboyer/foobar/baz
//
// Verify that prefix is path match and either the input is the same length as
// the match (in which case we know they're equal), or that the next character
// is a "/". (Import paths are defined to always use "/", not the OS-specific
// path separator.)
func isPathPrefix(path, pre string) bool {
	pathlen, prflen := len(path), len(pre)
	if pathlen < prflen || path[0:prflen] != pre {
		return false
	}

	return prflen == pathlen || strings.Index(path[prflen:], "/") == 0
}

// gvsubgraph is a graphviz subgraph with at least one node(package) in it.
type gvsubgraph struct {
	project  string   // Project root name of a project.
	packages []string // List of subpackages in the project.
	index    int      // Index of the subgraph cluster. This is used to refer the subgraph in the dot file.
	children []string // Dependencies of the project root package.
}

func (sg gvsubgraph) hash() uint32 {
	h := fnv.New32a()
	h.Write([]byte(sg.project))
	return h.Sum32()
}

// createSubgraph creates a graphviz subgraph with nodes in it. This should only
// be created when a project has more than one package. A single package project
// should be just a single node.
// First nodes are created using the provided packages and their imports. Then
// a subgraph is created with all the nodes in it.
func (g *graphviz) createSubgraph(project string, packages map[string][]string) {
	// If there's only a single package and that's the project root, do not
	// create a subgraph. Just create a node.
	if children, ok := packages[project]; ok && len(packages) == 1 {
		g.createNode(project, "", children)
		return
	}

	// Sort and use the packages for consistent output.
	pkgs := []gvnode{}

	for name, children := range packages {
		pkgs = append(pkgs, gvnode{project: name, children: children})
	}

	sort.Sort(byGvnode(pkgs))

	subgraphPkgs := []string{}
	rootChildren := []string{}
	for _, p := range pkgs {
		if p.project == project {
			// Do not create a separate node for the root package.
			rootChildren = append(rootChildren, p.children...)
			continue
		}
		g.createNode(p.project, "", p.children)
		subgraphPkgs = append(subgraphPkgs, p.project)
	}

	sg := &gvsubgraph{
		project:  project,
		packages: subgraphPkgs,
		index:    len(g.clusters),
		children: rootChildren,
	}

	g.h[project] = sg.hash()
	g.clusters[project] = sg
}

// sortCluster takes a map of all the clusters and returns a list of cluster
// names sorted by the cluster index.
func sortClusters(clusters map[string]*gvsubgraph) []*gvsubgraph {
	result := []*gvsubgraph{}
	for _, cluster := range clusters {
		result = append(result, cluster)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].index < result[j].index
	})
	return result
}
