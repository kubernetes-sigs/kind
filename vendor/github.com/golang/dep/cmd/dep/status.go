// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"text/template"

	"github.com/golang/dep"
	"github.com/golang/dep/gps"
	"github.com/golang/dep/gps/paths"
	"github.com/golang/dep/gps/verify"
	"github.com/pkg/errors"
)

const availableTemplateVariables = "ProjectRoot, Constraint, Version, Revision, Latest, and PackageCount."
const availableDefaultTemplateVariables = `.Projects[]{
	    .ProjectRoot,.Source,.Constraint,.PackageCount,.Packages[],
		.PruneOpts,.Digest,.Locked{.Branch,.Revision,.Version},
		.Latest{.Revision,.Version}
	},
	.Metadata{
	    .AnalyzerName,.AnalyzerVersion,.InputImports,.SolverName,
	    .SolverVersion
	}`

const statusShortHelp = `Report the status of the project's dependencies`
const statusLongHelp = `
With no arguments, print the status of each dependency of the project.

  PROJECT     Import path
  CONSTRAINT  Version constraint, from the manifest
  VERSION     Version chosen, from the lock
  REVISION    VCS revision of the chosen version
  LATEST      Latest VCS revision available
  PKGS USED   Number of packages from this project that are actually used

You may use the -f flag to create a custom format for the output of the
dep status command. The available fields you can utilize are as follows:
` + availableTemplateVariables + `

Status returns exit code zero if all dependencies are in a "good state".
`

const statusExamples = `
dep status

	Displays a table of the various dependencies in the project along with
	their properties such as the constraints they are bound by and the
	revision they are at.

dep status -detail

	Displays a detailed table of the dependencies in the project including
	the value of any source rules used and full list of packages used from
	each project (instead of simply a count). Text wrapping may make this
	output hard to read.

dep status -f='{{if eq .Constraint "master"}}{{.ProjectRoot}} {{end}}'

	Displays the list of package names constrained on the master branch.
	The -f flag allows you to use Go templates along with it's various
	constructs for formating output data. Available flags are as follows:
	` + availableTemplateVariables + `

dep status -detail -f='{{range $i, $p := .Projects}}{{if ne .Source "" -}}
	    {{- if $i}},{{end}}{{$p.ProjectRoot}}:{{$p.Source}}{{end}}{{end}}'

	Displays the package name and source for each package with a source
	rule defined, with a comma between each name-source pair.

	When used with -detail, the -f flag applies the supplied Go templates
	to the full output document, instead of to packages one at a time.
	Available flags are as follows: ` + availableDefaultTemplateVariables + `

dep status -json

	Displays the dependency information in JSON format as a list of
	project objects. Each project object contains keys which correspond
	to the table column names from the standard 'dep status' command.

Linux:   dep status -dot | dot -T png | display
MacOS:   dep status -dot | dot -T png | open -f -a /Applications/Preview.app
Windows: dep status -dot | dot -T png -o status.png; start status.png

	Generates a visual representation of the dependency tree using GraphViz.
	(Note: in order for this example to work you must first have graphviz
	installed on your system)

`

const (
	shortRev uint8 = iota
	longRev
)

var (
	errFailedUpdate        = errors.New("failed to fetch updates")
	errFailedListPkg       = errors.New("failed to list packages")
	errMultipleFailures    = errors.New("multiple sources of failure")
	errInputDigestMismatch = errors.New("input-digest mismatch")
)

func (cmd *statusCommand) Name() string      { return "status" }
func (cmd *statusCommand) Args() string      { return "[package...]" }
func (cmd *statusCommand) ShortHelp() string { return statusShortHelp }
func (cmd *statusCommand) LongHelp() string  { return statusLongHelp }
func (cmd *statusCommand) Hidden() bool      { return false }

func (cmd *statusCommand) Register(fs *flag.FlagSet) {
	fs.BoolVar(&cmd.examples, "examples", false, "print detailed usage examples")
	fs.BoolVar(&cmd.json, "json", false, "output in JSON format")
	fs.StringVar(&cmd.template, "f", "", "output in text/template format")
	fs.BoolVar(&cmd.lock, "lock", false, "output in the lock file format (assumes -detail)")
	fs.BoolVar(&cmd.dot, "dot", false, "output the dependency graph in GraphViz format")
	fs.BoolVar(&cmd.old, "old", false, "only show out-of-date dependencies")
	fs.BoolVar(&cmd.missing, "missing", false, "only show missing dependencies")
	fs.StringVar(&cmd.outFilePath, "out", "", "path to a file to which to write the output. Blank value will be ignored")
	fs.BoolVar(&cmd.detail, "detail", false, "include more detail in the chosen format")
}

type statusCommand struct {
	examples    bool
	json        bool
	template    string
	lock        bool
	output      string
	dot         bool
	old         bool
	missing     bool
	outFilePath string
	detail      bool
}

type outputter interface {
	BasicHeader() error
	BasicLine(*BasicStatus) error
	BasicFooter() error
	DetailHeader(*dep.SolveMeta) error
	DetailLine(*DetailStatus) error
	DetailFooter(*dep.SolveMeta) error
	MissingHeader() error
	MissingLine(*MissingStatus) error
	MissingFooter() error
}

// Only a subset of the outputters should be able to output old statuses.
type oldOutputter interface {
	OldHeader() error
	OldLine(*OldStatus) error
	OldFooter() error
}

type tableOutput struct{ w *tabwriter.Writer }

func (out *tableOutput) BasicHeader() error {
	_, err := fmt.Fprintf(out.w, "PROJECT\tCONSTRAINT\tVERSION\tREVISION\tLATEST\tPKGS USED\n")
	return err
}

func (out *tableOutput) BasicFooter() error {
	return out.w.Flush()
}

func (out *tableOutput) BasicLine(bs *BasicStatus) error {
	_, err := fmt.Fprintf(out.w,
		"%s\t%s\t%s\t%s\t%s\t%d\t\n",
		bs.ProjectRoot,
		bs.getConsolidatedConstraint(),
		formatVersion(bs.Version),
		formatVersion(bs.Revision),
		bs.getConsolidatedLatest(shortRev),
		bs.PackageCount,
	)
	return err
}

func (out *tableOutput) DetailHeader(metadata *dep.SolveMeta) error {
	_, err := fmt.Fprintf(out.w, "PROJECT\tSOURCE\tCONSTRAINT\tVERSION\tREVISION\tLATEST\tPKGS USED\n")
	return err
}

func (out *tableOutput) DetailFooter(metadata *dep.SolveMeta) error {
	return out.BasicFooter()
}

func (out *tableOutput) DetailLine(ds *DetailStatus) error {
	_, err := fmt.Fprintf(out.w,
		"%s\t%s\t%s\t%s\t%s\t%s\t[%s]\t\n",
		ds.ProjectRoot,
		ds.Source,
		ds.getConsolidatedConstraint(),
		formatVersion(ds.Version),
		formatVersion(ds.Revision),
		ds.getConsolidatedLatest(shortRev),
		strings.Join(ds.Packages, ", "),
	)
	return err
}

func (out *tableOutput) MissingHeader() error {
	_, err := fmt.Fprintln(out.w, "PROJECT\tMISSING PACKAGES")
	return err
}

func (out *tableOutput) MissingLine(ms *MissingStatus) error {
	_, err := fmt.Fprintf(out.w,
		"%s\t%s\t\n",
		ms.ProjectRoot,
		ms.MissingPackages,
	)
	return err
}

func (out *tableOutput) MissingFooter() error {
	return out.w.Flush()
}

func (out *tableOutput) OldHeader() error {
	_, err := fmt.Fprintf(out.w, "PROJECT\tCONSTRAINT\tREVISION\tLATEST\n")
	return err
}

func (out *tableOutput) OldLine(os *OldStatus) error {
	_, err := fmt.Fprintf(out.w,
		"%s\t%s\t%s\t%s\t\n",
		os.ProjectRoot,
		os.getConsolidatedConstraint(),
		formatVersion(os.Revision),
		os.getConsolidatedLatest(shortRev),
	)
	return err
}

func (out *tableOutput) OldFooter() error {
	return out.w.Flush()
}

type jsonOutput struct {
	w       io.Writer
	basic   []*rawStatus
	detail  []rawDetailProject
	missing []*MissingStatus
	old     []*rawOldStatus
}

func (out *jsonOutput) BasicHeader() error {
	out.basic = []*rawStatus{}
	return nil
}

func (out *jsonOutput) BasicFooter() error {
	return json.NewEncoder(out.w).Encode(out.basic)
}

func (out *jsonOutput) BasicLine(bs *BasicStatus) error {
	out.basic = append(out.basic, bs.marshalJSON())
	return nil
}

func (out *jsonOutput) DetailHeader(metadata *dep.SolveMeta) error {
	out.detail = []rawDetailProject{}
	return nil
}

func (out *jsonOutput) DetailFooter(metadata *dep.SolveMeta) error {
	doc := rawDetail{
		Projects: out.detail,
		Metadata: newRawMetadata(metadata),
	}

	return json.NewEncoder(out.w).Encode(doc)
}

func (out *jsonOutput) DetailLine(ds *DetailStatus) error {
	out.detail = append(out.detail, *ds.marshalJSON())
	return nil
}

func (out *jsonOutput) MissingHeader() error {
	out.missing = []*MissingStatus{}
	return nil
}

func (out *jsonOutput) MissingLine(ms *MissingStatus) error {
	out.missing = append(out.missing, ms)
	return nil
}

func (out *jsonOutput) MissingFooter() error {
	return json.NewEncoder(out.w).Encode(out.missing)
}

func (out *jsonOutput) OldHeader() error {
	out.old = []*rawOldStatus{}
	return nil
}

func (out *jsonOutput) OldLine(os *OldStatus) error {
	out.old = append(out.old, os.marshalJSON())
	return nil
}

func (out *jsonOutput) OldFooter() error {
	return json.NewEncoder(out.w).Encode(out.old)
}

type dotOutput struct {
	w io.Writer
	o string
	g *graphviz
	p *dep.Project
}

func (out *dotOutput) BasicHeader() error {
	out.g = new(graphviz).New()

	ptree := out.p.RootPackageTree
	// TODO(sdboyer) should be true, true, false, out.p.Manifest.IgnoredPackages()
	prm, _ := ptree.ToReachMap(true, false, false, nil)

	out.g.createNode(string(out.p.ImportRoot), "", prm.FlattenFn(paths.IsStandardImportPath))

	return nil
}

func (out *dotOutput) BasicFooter() error {
	gvo := out.g.output("")
	_, err := fmt.Fprintf(out.w, gvo.String())
	return err
}

func (out *dotOutput) BasicLine(bs *BasicStatus) error {
	out.g.createNode(bs.ProjectRoot, bs.getConsolidatedVersion(), bs.Children)
	return nil
}

func (out *dotOutput) DetailHeader(metadata *dep.SolveMeta) error {
	return out.BasicHeader()
}

func (out *dotOutput) DetailFooter(metadata *dep.SolveMeta) error {
	return out.BasicFooter()
}

func (out *dotOutput) DetailLine(ds *DetailStatus) error {
	return out.BasicLine(&ds.BasicStatus)
}

func (out *dotOutput) MissingHeader() error                { return nil }
func (out *dotOutput) MissingLine(ms *MissingStatus) error { return nil }
func (out *dotOutput) MissingFooter() error                { return nil }

type templateOutput struct {
	w      io.Writer
	tmpl   *template.Template
	detail []rawDetailProject
}

func (out *templateOutput) BasicHeader() error { return nil }
func (out *templateOutput) BasicFooter() error { return nil }
func (out *templateOutput) BasicLine(bs *BasicStatus) error {
	data := rawStatus{
		ProjectRoot:  bs.ProjectRoot,
		Constraint:   bs.getConsolidatedConstraint(),
		Version:      bs.getConsolidatedVersion(),
		Revision:     bs.Revision.String(),
		Latest:       bs.getConsolidatedLatest(shortRev),
		PackageCount: bs.PackageCount,
	}
	return out.tmpl.Execute(out.w, data)
}

func (out *templateOutput) DetailHeader(metadata *dep.SolveMeta) error {
	out.detail = []rawDetailProject{}

	return nil
}

func (out *templateOutput) DetailFooter(metadata *dep.SolveMeta) error {
	raw := rawDetail{
		Projects: out.detail,
		Metadata: newRawMetadata(metadata),
	}

	return out.tmpl.Execute(out.w, raw)
}

func (out *templateOutput) DetailLine(ds *DetailStatus) error {
	data := rawDetailProject{
		ProjectRoot:  ds.ProjectRoot,
		Constraint:   ds.getConsolidatedConstraint(),
		Locked:       formatDetailVersion(ds.Version, ds.Revision),
		Latest:       formatDetailLatestVersion(ds.Latest, ds.hasError),
		PruneOpts:    ds.getPruneOpts(),
		Digest:       ds.Digest.String(),
		PackageCount: ds.PackageCount,
		Source:       ds.Source,
		Packages:     ds.Packages,
	}

	out.detail = append(out.detail, data)

	return nil
}

func (out *templateOutput) OldHeader() error { return nil }
func (out *templateOutput) OldFooter() error { return nil }
func (out *templateOutput) OldLine(os *OldStatus) error {
	return out.tmpl.Execute(out.w, os)
}

func (out *templateOutput) MissingHeader() error { return nil }
func (out *templateOutput) MissingFooter() error { return nil }
func (out *templateOutput) MissingLine(ms *MissingStatus) error {
	return out.tmpl.Execute(out.w, ms)
}

func (cmd *statusCommand) Run(ctx *dep.Ctx, args []string) error {
	if cmd.examples {
		ctx.Err.Println(strings.TrimSpace(statusExamples))
		return nil
	}

	if err := cmd.validateFlags(); err != nil {
		return err
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

	if err := dep.ValidateProjectRoots(ctx, p.Manifest, sm); err != nil {
		return err
	}

	var buf bytes.Buffer
	var out outputter
	switch {
	case cmd.missing:
		return errors.Errorf("not implemented")
	case cmd.json:
		out = &jsonOutput{
			w: &buf,
		}
	case cmd.dot:
		out = &dotOutput{
			p: p,
			o: cmd.output,
			w: &buf,
		}
	case cmd.template != "":
		tmpl, err := parseStatusTemplate(cmd.template)
		if err != nil {
			return err
		}
		out = &templateOutput{
			w:    &buf,
			tmpl: tmpl,
		}
	case cmd.lock:
		tmpl, err := parseStatusTemplate(statusLockTemplate)
		if err != nil {
			return err
		}
		out = &templateOutput{
			w:    &buf,
			tmpl: tmpl,
		}
	default:
		out = &tableOutput{
			w: tabwriter.NewWriter(&buf, 0, 4, 2, ' ', 0),
		}
	}

	// Check if the lock file exists.
	if p.Lock == nil {
		return errors.Errorf("no Gopkg.lock found. Run `dep ensure` to generate lock file")
	}

	if cmd.old {
		if _, ok := out.(oldOutputter); !ok {
			return errors.Errorf("invalid output format used")
		}
		err = cmd.runOld(ctx, out.(oldOutputter), p, sm)
		ctx.Out.Print(buf.String())
		return err
	}

	_, errCount, runerr := cmd.runStatusAll(ctx, out, p, sm)
	if runerr != nil {
		switch runerr {
		case errFailedUpdate:
			// Print the help when in non-verbose mode
			if !ctx.Verbose {
				ctx.Out.Printf("The status of %d projects are unknown due to errors. Rerun with `-v` flag to see details.\n", errCount)
			}
		case errInputDigestMismatch:
			ctx.Err.Printf("Gopkg.lock is out of sync with imports and/or Gopkg.toml. Run `dep check` for details.\n")
		default:
			return runerr
		}
	}

	if cmd.outFilePath == "" {
		// Print the status output
		ctx.Out.Print(buf.String())
	} else {
		file, err := os.Create(cmd.outFilePath)
		if err != nil {
			return fmt.Errorf("error creating output file: %v", err)
		}

		defer file.Close()
		if _, err := io.Copy(file, bytes.NewReader(buf.Bytes())); err != nil {
			return fmt.Errorf("error writing output file: %v", err)
		}
	}

	return runerr
}

func (cmd *statusCommand) validateFlags() error {
	// Operating mode flags.
	var opModes []string

	if cmd.old {
		opModes = append(opModes, "-old")
	}

	if cmd.missing {
		opModes = append(opModes, "-missing")
	}

	if cmd.detail {
		opModes = append(opModes, "-detail")
	}

	// Check if any other flags are passed with -dot.
	if cmd.dot {
		if cmd.template != "" {
			return errors.New("cannot pass template string with -dot")
		}

		if cmd.json {
			return errors.New("cannot pass multiple output format flags")
		}

		if len(opModes) > 0 {
			return errors.New("-dot generates dependency graph; cannot pass other flags")
		}
	}

	if cmd.lock {
		if cmd.template != "" {
			return errors.New("cannot pass template string with -lock")
		}

		if !cmd.detail {
			cmd.detail = true
		}
	}

	if len(opModes) > 1 {
		// List the flags because which flags are for operation mode might not
		// be apparent to the users.
		return errors.Wrapf(errors.New("cannot pass multiple operating mode flags"), "%v", opModes)
	}

	return nil
}

// OldStatus contains information about all the out of date packages in a project.
type OldStatus struct {
	ProjectRoot string
	Constraint  gps.Constraint
	Revision    gps.Revision
	Latest      gps.Version
}

type rawOldStatus struct {
	ProjectRoot, Constraint, Revision, Latest string
}

func (os OldStatus) getConsolidatedConstraint() string {
	var constraint string
	if os.Constraint != nil {
		if v, ok := os.Constraint.(gps.Version); ok {
			constraint = formatVersion(v)
		} else {
			constraint = os.Constraint.String()
		}
	}
	return constraint
}

func (os OldStatus) getConsolidatedLatest(revSize uint8) string {
	latest := ""
	if os.Latest != nil {
		switch revSize {
		case shortRev:
			latest = formatVersion(os.Latest)
		case longRev:
			latest = os.Latest.String()
		}
	}
	return latest
}

func (os OldStatus) marshalJSON() *rawOldStatus {
	return &rawOldStatus{
		ProjectRoot: os.ProjectRoot,
		Constraint:  os.getConsolidatedConstraint(),
		Revision:    string(os.Revision),
		Latest:      os.getConsolidatedLatest(longRev),
	}
}

func (cmd *statusCommand) runOld(ctx *dep.Ctx, out oldOutputter, p *dep.Project, sm gps.SourceManager) error {
	// While the network churns on ListVersions() requests, statically analyze
	// code from the current project.
	ptree := p.RootPackageTree

	// Set up a solver in order to check the InputHash.
	params := gps.SolveParameters{
		ProjectAnalyzer: dep.Analyzer{},
		RootDir:         p.AbsRoot,
		RootPackageTree: ptree,
		Manifest:        p.Manifest,
		// Locks aren't a part of the input hash check, so we can omit it.
	}

	logger := ctx.Err
	if ctx.Verbose {
		params.TraceLogger = ctx.Err
	} else {
		logger = log.New(ioutil.Discard, "", 0)
	}

	// Check update for all the projects.
	params.ChangeAll = true

	solver, err := gps.Prepare(params, sm)
	if err != nil {
		return errors.Wrap(err, "fastpath solver prepare")
	}

	logger.Println("Solving dependency graph to determine which dependencies can be updated.")
	solution, err := solver.Solve(context.TODO())
	if err != nil {
		return errors.Wrap(err, "runOld")
	}

	var oldStatuses []OldStatus
	solutionProjects := solution.Projects()

	for _, proj := range p.Lock.Projects() {
		for _, sProj := range solutionProjects {
			// Look for the same project in solution and lock.
			if sProj.Ident().ProjectRoot != proj.Ident().ProjectRoot {
				continue
			}

			// If revisions are not the same then it is old and we should display it.
			latestRev, _, _ := gps.VersionComponentStrings(sProj.Version())
			atRev, _, _ := gps.VersionComponentStrings(proj.Version())
			if atRev == latestRev {
				continue
			}

			var constraint gps.Constraint
			// Getting Constraint.
			if pp, has := p.Manifest.Ovr[proj.Ident().ProjectRoot]; has && pp.Constraint != nil {
				// manifest has override for project.
				constraint = pp.Constraint
			} else if pp, has := p.Manifest.Constraints[proj.Ident().ProjectRoot]; has && pp.Constraint != nil {
				// manifest has normal constraint.
				constraint = pp.Constraint
			} else {
				// No constraint exists. No need to worry about displaying it.
				continue
			}

			// Generate the old status data and append it.
			os := OldStatus{
				ProjectRoot: proj.Ident().String(),
				Revision:    gps.Revision(atRev),
				Latest:      gps.Revision(latestRev),
				Constraint:  constraint,
			}
			oldStatuses = append(oldStatuses, os)
		}
	}

	out.OldHeader()
	for _, ostat := range oldStatuses {
		out.OldLine(&ostat)
	}
	out.OldFooter()

	return nil
}

type rawStatus struct {
	ProjectRoot  string
	Constraint   string
	Version      string
	Revision     string
	Latest       string
	PackageCount int
}

// rawDetail is is additional information used for the status when the
// -detail flag is specified
type rawDetail struct {
	Projects []rawDetailProject
	Metadata rawDetailMetadata
}

type rawDetailVersion struct {
	Revision string `json:"Revision,omitempty"`
	Version  string `json:"Version,omitempty"`
	Branch   string `json:"Branch,omitempty"`
}

type rawDetailProject struct {
	ProjectRoot  string
	Packages     []string
	Locked       rawDetailVersion
	Latest       rawDetailVersion
	PruneOpts    string
	Digest       string
	Source       string `json:"Source,omitempty"`
	Constraint   string
	PackageCount int
}

type rawDetailMetadata struct {
	AnalyzerName    string
	AnalyzerVersion int
	InputsDigest    string // deprecated
	InputImports    []string
	SolverName      string
	SolverVersion   int
}

func newRawMetadata(metadata *dep.SolveMeta) rawDetailMetadata {
	if metadata == nil {
		return rawDetailMetadata{}
	}

	return rawDetailMetadata{
		AnalyzerName:    metadata.AnalyzerName,
		AnalyzerVersion: metadata.AnalyzerVersion,
		InputImports:    metadata.InputImports,
		SolverName:      metadata.SolverName,
		SolverVersion:   metadata.SolverVersion,
	}
}

// BasicStatus contains all the information reported about a single dependency
// in the summary/list status output mode.
type BasicStatus struct {
	ProjectRoot  string
	Children     []string
	Constraint   gps.Constraint
	Version      gps.UnpairedVersion
	Revision     gps.Revision
	Latest       gps.Version
	PackageCount int
	hasOverride  bool
	hasError     bool
}

// DetailStatus contains all information reported about a single dependency
// in the detailed status output mode. The included information matches the
// information included about a a project in a lock file.
type DetailStatus struct {
	BasicStatus
	Packages  []string
	Source    string
	PruneOpts gps.PruneOptions
	Digest    verify.VersionedDigest
}

func (bs *BasicStatus) getConsolidatedConstraint() string {
	var constraint string
	if bs.Constraint != nil {
		if v, ok := bs.Constraint.(gps.Version); ok {
			constraint = formatVersion(v)
		} else {
			constraint = bs.Constraint.String()
		}
	}

	if bs.hasOverride {
		constraint += " (override)"
	}

	return constraint
}

func (bs *BasicStatus) getConsolidatedVersion() string {
	version := formatVersion(bs.Revision)
	if bs.Version != nil {
		version = formatVersion(bs.Version)
	}
	return version
}

func (bs *BasicStatus) getConsolidatedLatest(revSize uint8) string {
	latest := ""
	if bs.Latest != nil {
		switch revSize {
		case shortRev:
			latest = formatVersion(bs.Latest)
		case longRev:
			latest = bs.Latest.String()
		}
	}

	if bs.hasError {
		latest += "unknown"
	}

	return latest
}

func (ds *DetailStatus) getPruneOpts() string {
	return (ds.PruneOpts & ^gps.PruneNestedVendorDirs).String()
}

func (bs *BasicStatus) marshalJSON() *rawStatus {
	return &rawStatus{
		ProjectRoot:  bs.ProjectRoot,
		Constraint:   bs.getConsolidatedConstraint(),
		Version:      formatVersion(bs.Version),
		Revision:     string(bs.Revision),
		Latest:       bs.getConsolidatedLatest(longRev),
		PackageCount: bs.PackageCount,
	}
}

func (ds *DetailStatus) marshalJSON() *rawDetailProject {
	rawStatus := ds.BasicStatus.marshalJSON()

	return &rawDetailProject{
		ProjectRoot:  rawStatus.ProjectRoot,
		Constraint:   rawStatus.Constraint,
		Locked:       formatDetailVersion(ds.Version, ds.Revision),
		Latest:       formatDetailLatestVersion(ds.Latest, ds.hasError),
		PruneOpts:    ds.getPruneOpts(),
		Digest:       ds.Digest.String(),
		Source:       ds.Source,
		Packages:     ds.Packages,
		PackageCount: ds.PackageCount,
	}
}

// MissingStatus contains information about all the missing packages in a project.
type MissingStatus struct {
	ProjectRoot     string
	MissingPackages []string
}

func (cmd *statusCommand) runStatusAll(ctx *dep.Ctx, out outputter, p *dep.Project, sm gps.SourceManager) (hasMissingPkgs bool, errCount int, err error) {
	// While the network churns on ListVersions() requests, statically analyze
	// code from the current project.
	ptree := p.RootPackageTree

	// Set up a solver in order to check the InputHash.
	params := gps.SolveParameters{
		ProjectAnalyzer: dep.Analyzer{},
		RootDir:         p.AbsRoot,
		RootPackageTree: ptree,
		Manifest:        p.Manifest,
		// Locks aren't a part of the input hash check, so we can omit it.
	}

	logger := ctx.Err
	if ctx.Verbose {
		params.TraceLogger = ctx.Err
	} else {
		logger = log.New(ioutil.Discard, "", 0)
	}

	if err := ctx.ValidateParams(sm, params); err != nil {
		return false, 0, err
	}

	// Errors while collecting constraints should not fail the whole status run.
	// It should count the error and tell the user about incomplete results.
	cm, ccerrs := collectConstraints(ctx, p, sm)
	if len(ccerrs) > 0 {
		errCount += len(ccerrs)
	}

	// Get the project list and sort it so that the printed output users see is
	// deterministically ordered. (This may be superfluous if the lock is always
	// written in alpha order, but it doesn't hurt to double down.)
	slp := p.Lock.Projects()
	sort.Slice(slp, func(i, j int) bool {
		return slp[i].Ident().Less(slp[j].Ident())
	})
	slcp := p.ChangedLock.Projects()
	sort.Slice(slcp, func(i, j int) bool {
		return slcp[i].Ident().Less(slcp[j].Ident())
	})

	lsat := verify.LockSatisfiesInputs(p.Lock, p.Manifest, params.RootPackageTree)
	if lsat.Satisfied() {
		// If the lock satisfies the inputs, we're guaranteed (barring manual
		// meddling, about which we can do nothing) that the lock is a
		// transitively complete picture of all deps. That eliminates the need
		// for some checks.

		logger.Println("Checking upstream projects:")

		// DetailStatus channel to collect all the DetailStatus.
		dsCh := make(chan *DetailStatus, len(slp))

		// Error channels to collect different errors.
		errListPkgCh := make(chan error, len(slp))
		errListVerCh := make(chan error, len(slp))

		var wg sync.WaitGroup

		for i, proj := range slp {
			wg.Add(1)
			logger.Printf("(%d/%d) %s\n", i+1, len(slp), proj.Ident().ProjectRoot)

			go func(proj verify.VerifiableProject) {
				bs := BasicStatus{
					ProjectRoot:  string(proj.Ident().ProjectRoot),
					PackageCount: len(proj.Packages()),
				}

				// Get children only for specific outputers
				// in order to avoid slower status process.
				switch out.(type) {
				case *dotOutput:
					ptr, err := sm.ListPackages(proj.Ident(), proj.Version())

					if err != nil {
						bs.hasError = true
						errListPkgCh <- err
					}

					prm, _ := ptr.ToReachMap(true, true, false, p.Manifest.IgnoredPackages())
					bs.Children = prm.FlattenFn(paths.IsStandardImportPath)
				}

				// Split apart the version from the lock into its constituent parts.
				switch tv := proj.Version().(type) {
				case gps.UnpairedVersion:
					bs.Version = tv
				case gps.Revision:
					bs.Revision = tv
				case gps.PairedVersion:
					bs.Version = tv.Unpair()
					bs.Revision = tv.Revision()
				}

				// Check if the manifest has an override for this project. If so,
				// set that as the constraint.
				if pp, has := p.Manifest.Ovr[proj.Ident().ProjectRoot]; has && pp.Constraint != nil {
					bs.hasOverride = true
					bs.Constraint = pp.Constraint
				} else if pp, has := p.Manifest.Constraints[proj.Ident().ProjectRoot]; has && pp.Constraint != nil {
					// If the manifest has a constraint then set that as the constraint.
					bs.Constraint = pp.Constraint
				} else {
					bs.Constraint = gps.Any()
					for _, c := range cm[bs.ProjectRoot] {
						bs.Constraint = c.Constraint.Intersect(bs.Constraint)
					}
				}

				// Only if we have a non-rev and non-plain version do/can we display
				// anything wrt the version's updateability.
				if bs.Version != nil && bs.Version.Type() != gps.IsVersion {
					c, has := p.Manifest.Constraints[proj.Ident().ProjectRoot]
					if !has {
						// Get constraint for locked project
						for _, lockedP := range p.Lock.P {
							if lockedP.Ident().ProjectRoot == proj.Ident().ProjectRoot {
								// Use the unpaired version as the constraint for checking updates.
								c.Constraint = bs.Version
							}
						}
					}
					// TODO: This constraint is only the constraint imposed by the
					// current project, not by any transitive deps. As a result,
					// transitive project deps will always show "any" here.
					bs.Constraint = c.Constraint

					vl, err := sm.ListVersions(proj.Ident())
					if err == nil {
						gps.SortPairedForUpgrade(vl)

						for _, v := range vl {
							// Because we've sorted the version list for
							// upgrade, the first version we encounter that
							// matches our constraint will be what we want.
							if c.Constraint.Matches(v) {
								// Latest should be of the same type as the Version.
								if bs.Version.Type() == gps.IsSemver {
									bs.Latest = v
								} else {
									bs.Latest = v.Revision()
								}
								break
							}
						}
					} else {
						// Failed to fetch version list (could happen due to
						// network issue).
						bs.hasError = true
						errListVerCh <- err
					}
				}

				ds := DetailStatus{
					BasicStatus: bs,
				}

				if cmd.detail {
					ds.Source = proj.Ident().Source
					ds.Packages = proj.Packages()
					ds.PruneOpts = proj.PruneOpts
					ds.Digest = proj.Digest
				}

				dsCh <- &ds

				wg.Done()
			}(proj.(verify.VerifiableProject))
		}

		wg.Wait()
		close(dsCh)
		close(errListPkgCh)
		close(errListVerCh)

		// Newline after printing the status progress output.
		logger.Println()

		// List Packages errors. This would happen only for dot output.
		if len(errListPkgCh) > 0 {
			err = errFailedListPkg
			if ctx.Verbose {
				for err := range errListPkgCh {
					ctx.Err.Println(err.Error())
				}
				ctx.Err.Println()
			}
		}

		// List Version errors.
		if len(errListVerCh) > 0 {
			if err == nil {
				err = errFailedUpdate
			} else {
				err = errMultipleFailures
			}

			// Count ListVersions error because we get partial results when
			// this happens.
			errCount += len(errListVerCh)
			if ctx.Verbose {
				for err := range errListVerCh {
					ctx.Err.Println(err.Error())
				}
				ctx.Err.Println()
			}
		}

		if cmd.detail {
			// A map of ProjectRoot and *DetailStatus. This is used in maintain the
			// order of DetailStatus in output by collecting all the DetailStatus and
			// then using them in order.
			dsMap := make(map[string]*DetailStatus)
			for ds := range dsCh {
				dsMap[ds.ProjectRoot] = ds
			}

			if err := detailOutputAll(out, slp, dsMap, &p.Lock.SolveMeta); err != nil {
				return false, 0, err
			}
		} else {
			// A map of ProjectRoot and *BasicStatus. This is used in maintain the
			// order of BasicStatus in output by collecting all the BasicStatus and
			// then using them in order.
			bsMap := make(map[string]*BasicStatus)
			for bs := range dsCh {
				bsMap[bs.ProjectRoot] = &bs.BasicStatus
			}

			if err := basicOutputAll(out, slp, bsMap); err != nil {
				return false, 0, err
			}
		}

		return false, errCount, err
	}

	rm, _ := ptree.ToReachMap(true, true, false, p.Manifest.IgnoredPackages())

	external := rm.FlattenFn(paths.IsStandardImportPath)
	roots := make(map[gps.ProjectRoot][]string, len(external))

	type fail struct {
		ex  string
		err error
	}
	var errs []fail
	for _, e := range external {
		root, err := sm.DeduceProjectRoot(e)
		if err != nil {
			errs = append(errs, fail{
				ex:  e,
				err: err,
			})
			continue
		}

		roots[root] = append(roots[root], e)
	}

	if len(errs) != 0 {
		// TODO this is just a fix quick so staticcheck doesn't complain.
		// Visually reconciling failure to deduce project roots with the rest of
		// the mismatch output is a larger problem.
		ctx.Err.Printf("Failed to deduce project roots for import paths:\n")
		for _, fail := range errs {
			ctx.Err.Printf("\t%s: %s\n", fail.ex, fail.err.Error())
		}

		return false, 0, errors.New("address issues with undeducible import paths to get more status information")
	}

	if err = out.MissingHeader(); err != nil {
		return false, 0, err
	}

outer:
	for root, pkgs := range roots {
		// TODO also handle the case where the project is present, but there
		// are items missing from just the package list
		for _, lp := range slp {
			if lp.Ident().ProjectRoot == root {
				continue outer
			}
		}

		hasMissingPkgs = true
		err := out.MissingLine(&MissingStatus{ProjectRoot: string(root), MissingPackages: pkgs})
		if err != nil {
			return false, 0, err
		}
	}
	if err = out.MissingFooter(); err != nil {
		return false, 0, err
	}

	// We are here because of an input-digest mismatch. Return error.
	return hasMissingPkgs, 0, errInputDigestMismatch
}

// basicOutputAll takes an outputter, a project list, and a map of ProjectRoot to *BasicStatus and
// uses the outputter to output basic header, body lines (in the order of the project list), and
// footer based on the project information.
func basicOutputAll(out outputter, slp []gps.LockedProject, bsMap map[string]*BasicStatus) (err error) {
	if err := out.BasicHeader(); err != nil {
		return err
	}

	// Use the collected BasicStatus in outputter.
	for _, proj := range slp {
		if err := out.BasicLine(bsMap[string(proj.Ident().ProjectRoot)]); err != nil {
			return err
		}
	}

	return out.BasicFooter()
}

// detailOutputAll takes an outputter, a project list, and a map of ProjectRoot to *DetailStatus and
// uses the outputter to output detailed header, body lines (in the order of the project list), and
// footer based on the project information.
func detailOutputAll(out outputter, slp []gps.LockedProject, dsMap map[string]*DetailStatus, metadata *dep.SolveMeta) (err error) {
	if err := out.DetailHeader(metadata); err != nil {
		return err
	}

	// Use the collected BasicStatus in outputter.
	for _, proj := range slp {
		if err := out.DetailLine(dsMap[string(proj.Ident().ProjectRoot)]); err != nil {
			return err
		}
	}

	return out.DetailFooter(metadata)
}

func formatVersion(v gps.Version) string {
	if v == nil {
		return ""
	}
	switch v.Type() {
	case gps.IsBranch:
		return "branch " + v.String()
	case gps.IsRevision:
		r := v.String()
		if len(r) > 7 {
			r = r[:7]
		}
		return r
	}
	return v.String()
}

func formatDetailVersion(v gps.Version, r gps.Revision) rawDetailVersion {
	if v == nil {
		return rawDetailVersion{
			Revision: r.String(),
		}
	}
	switch v.Type() {
	case gps.IsBranch:
		return rawDetailVersion{
			Branch:   v.String(),
			Revision: r.String(),
		}
	case gps.IsRevision:
		return rawDetailVersion{
			Revision: v.String(),
		}
	}

	return rawDetailVersion{
		Version:  v.String(),
		Revision: r.String(),
	}
}

func formatDetailLatestVersion(v gps.Version, hasError bool) rawDetailVersion {
	if hasError {
		return rawDetailVersion{
			Revision: "unknown",
		}
	}

	return formatDetailVersion(v, "")
}

// projectConstraint stores ProjectRoot and Constraint for that project.
type projectConstraint struct {
	Project    gps.ProjectRoot
	Constraint gps.Constraint
}

// constraintsCollection is a map of ProjectRoot(dependency) and a collection of
// projectConstraint for the dependencies. This can be used to find constraints
// on a dependency and the projects that apply those constraints.
type constraintsCollection map[string][]projectConstraint

// collectConstraints collects constraints declared by all the dependencies and
// constraints from the root project. It returns constraintsCollection and
// a slice of errors encountered while collecting the constraints, if any.
func collectConstraints(ctx *dep.Ctx, p *dep.Project, sm gps.SourceManager) (constraintsCollection, []error) {
	logger := ctx.Err
	if !ctx.Verbose {
		logger = log.New(ioutil.Discard, "", 0)
	}

	logger.Println("Collecting project constraints:")

	var mutex sync.Mutex
	constraintCollection := make(constraintsCollection)

	// Collect the complete set of direct project dependencies, incorporating
	// requireds and ignores appropriately.
	directDeps, err := p.GetDirectDependencyNames(sm)
	if err != nil {
		// Return empty collection, not nil, if we fail here.
		return constraintCollection, []error{errors.Wrap(err, "failed to get direct dependencies")}
	}

	// Create a root analyzer.
	rootAnalyzer := newRootAnalyzer(true, ctx, directDeps, sm)

	lp := p.Lock.Projects()

	// Channel for receiving all the errors.
	errCh := make(chan error, len(lp))

	var wg sync.WaitGroup

	// Iterate through the locked projects and collect constraints of all the projects.
	for i, proj := range lp {
		wg.Add(1)
		logger.Printf("(%d/%d) %s\n", i+1, len(lp), proj.Ident().ProjectRoot)

		go func(proj gps.LockedProject) {
			defer wg.Done()

			manifest, _, err := sm.GetManifestAndLock(proj.Ident(), proj.Version(), rootAnalyzer)
			if err != nil {
				errCh <- errors.Wrap(err, "error getting manifest and lock")
				return
			}

			// Get project constraints.
			pc := manifest.DependencyConstraints()

			// Obtain a lock for constraintCollection.
			mutex.Lock()
			defer mutex.Unlock()
			// Iterate through the project constraints to get individual dependency
			// project and constraint values.
			for pr, pp := range pc {
				// Check if the project constraint is imported in the root project
				if _, ok := directDeps[pr]; !ok {
					continue
				}

				tempCC := append(
					constraintCollection[string(pr)],
					projectConstraint{proj.Ident().ProjectRoot, pp.Constraint},
				)

				// Sort the inner projectConstraint slice by Project string.
				// Required for consistent returned value.
				sort.Sort(byProject(tempCC))
				constraintCollection[string(pr)] = tempCC
			}
		}(proj)
	}

	wg.Wait()
	close(errCh)

	var errs []error
	if len(errCh) > 0 {
		for e := range errCh {
			errs = append(errs, e)
			logger.Println(e.Error())
		}
	}

	// Incorporate constraints set in the manifest of the root project.
	if p.Manifest != nil {

		// Iterate through constraints in the manifest, append if it is a
		// direct dependency
		for pr, pp := range p.Manifest.Constraints {
			if _, ok := directDeps[pr]; !ok {
				continue
			}

			// Mark constraints coming from the manifest as "root"
			tempCC := append(
				constraintCollection[string(pr)],
				projectConstraint{"root", pp.Constraint},
			)

			// Sort the inner projectConstraint slice by Project string.
			// Required for consistent returned value.
			sort.Sort(byProject(tempCC))
			constraintCollection[string(pr)] = tempCC
		}
	}

	return constraintCollection, errs
}

type byProject []projectConstraint

func (p byProject) Len() int           { return len(p) }
func (p byProject) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p byProject) Less(i, j int) bool { return p[i].Project < p[j].Project }

func parseStatusTemplate(format string) (*template.Template, error) {
	tmpl, err := template.New("status").Funcs(template.FuncMap{
		"dec": func(i int) int {
			return i - 1
		},
		"tomlStrSplit": tomlStrSplit,
		"tomlStrSplit2": func(strlist []string, level int) string {
			// Hardcode to two spaces.
			inbracket, inp := strings.Repeat("  ", level), strings.Repeat("  ", level+1)
			switch len(strlist) {
			case 0:
				return "[]"
			case 1:
				return fmt.Sprintf("[\"%s\"]", strlist[0])
			default:
				var buf bytes.Buffer

				fmt.Fprintf(&buf, "[\n")
				for _, str := range strlist {
					fmt.Fprintf(&buf, "%s\"%s\",\n", inp, str)
				}
				fmt.Fprintf(&buf, "%s]", inbracket)

				return buf.String()
			}
		},
	}).Parse(format)

	return tmpl, err
}

func tomlStrSplit(strlist []string) string {
	switch len(strlist) {
	case 0:
		return "[]"
	case 1:
		return fmt.Sprintf("[\"%s\"]", strlist[0])
	default:
		var buf bytes.Buffer

		// Hardcode to two spaces.
		fmt.Fprintf(&buf, "[\n")
		for _, str := range strlist {
			fmt.Fprintf(&buf, "    \"%s\",\n", str)
		}
		fmt.Fprintf(&buf, "  ]")

		return buf.String()
	}
}

const statusLockTemplate = `# This file is autogenerated, do not edit; changes may be undone by the next 'dep ensure'.


{{range $p := .Projects}}[[projects]]
  {{- if $p.Locked.Branch}}
  branch = "{{$p.Locked.Branch}}"
  {{- end}}
  digest = "{{$p.Digest}}"
  name = "{{$p.ProjectRoot}}"
  packages = {{(tomlStrSplit $p.Packages)}}
  pruneopts = "{{$p.PruneOpts}}"
  revision = "{{$p.Locked.Revision}}"
  {{- if $p.Source}}
  source = "{{$p.Source}}"
  {{- end}}
  {{- if $p.Locked.Version}}
  version = "{{$p.Locked.Version}}"
  {{- end}}

{{end}}[solve-meta]
  analyzer-name = "{{.Metadata.AnalyzerName}}"
  analyzer-version = {{.Metadata.AnalyzerVersion}}
  input-imports = {{(tomlStrSplit .Metadata.InputImports)}}
  solver-name = "{{.Metadata.SolverName}}"
  solver-version = {{.Metadata.SolverVersion}}
`
