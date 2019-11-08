package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"os"
	"path"
	"runtime"
	"runtime/debug"
	"sort"

	"github.com/dmgk/portgrep/formatter"
	"github.com/dmgk/portgrep/grep"
	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
)

func main() {
	if flag.NFlag() == 0 {
		flag.Usage()
		os.Exit(0)
	}

	if flagVersion {
		fmt.Fprintln(os.Stderr, version)
		os.Exit(0)
	}

	var err error

	if flagSort {
		err = runSorted()
	} else {
		err = runUnsorted()
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(0)
	}
}

func runSorted() error {
	rxs, err := queries.compile()
	if err != nil {
		return err
	}

	type r struct {
		path    string
		matches grep.Matches
	}
	var rr []*r

	fn := func(path string, matches grep.Matches, err error) error {
		if err != nil {
			return err
		}
		rr = append(rr, &r{path, matches})
		return nil
	}

	err = grep.Grep(flagPortsRoot, rxs, fn, runtime.NumCPU())
	if err != nil {
		return err
	}

	sort.Slice(rr, func(i, j int) bool {
		return rr[i].path < rr[j].path
	})

	for _, r := range rr {
		if err := form.Format(r.path, r.matches); err != nil {
			return err
		}
	}

	return nil
}

func runUnsorted() error {
	rxs, err := queries.compile()
	if err != nil {
		return err
	}

	fn := func(path string, matches grep.Matches, err error) error {
		if err != nil {
			return err
		}
		return form.Format(path, matches)
	}

	return grep.Grep(flagPortsRoot, rxs, fn, runtime.NumCPU())
}

var form formatter.Formatter

func initFormatter() {
	var w io.Writer = os.Stdout
	flags := formatter.Fdefaults
	term := isatty.IsTerminal(os.Stdout.Fd())

	if flagColorMode == colorModeAlways || (term && flagColorMode == colorModeAuto) {
		w = colorable.NewColorableStdout()
		flags |= formatter.Fcolor
	}

	if flagOriginsSingleLine {
		flags |= formatter.ForiginsSingleLine
	}
	if flagOriginOnly {
		flags |= formatter.ForiginsOnly
	}

	form = formatter.NewText(w, flagPortsRoot, flags)
	if !flagNoIndent {
		form.SetIndent("\t")
	}
}

var usageTemplate = template.Must(template.New("Usage").Parse(`Usage: {{.basename}} <options>

Global options:
  -C mode    colorized output mode: [auto|never|always] (default: {{.colorMode}})
  -R path    ports tree root (default: {{.portsRoot}})
  -v         show version

Format options:
  -1         output origins in a single line (implies -o)
  -o         output origins only
  -s         sort results by origin
  -T         do not indent results

Search options:
  -x         treat query as a regular expression
  -d  query  search by *_DEPENDS
  -db query  search by BUILD_DEPENDS
  -dl query  search by LIB_DEPENDS
  -dr query  search by RUN_DEPENDS
  -m  query  search by MAINTAINER
  -u  query  search by USES
`))

const (
	colorModeAuto   = "auto"
	colorModeAlways = "always"
	colorModeNever  = "never"
)

type queryFlag struct {
	name string
	kind int
	val  string
}

type queryFlags []*queryFlag

func (qf queryFlags) any() bool {
	for _, q := range qf {
		if q.val != "" {
			return true
		}
	}
	return false
}

func (qf queryFlags) addFlags() {
	for _, q := range qf {
		flag.StringVar(&q.val, q.name, "", "")
	}
}

func (qf queryFlags) compile() ([]*grep.Regexp, error) {
	var res []*grep.Regexp
	for _, q := range qf {
		if q.val == "" {
			continue
		}
		re, err := grep.Compile(q.kind, q.val, flagRegexp)
		if err != nil {
			return nil, err
		}
		res = append(res, re)
	}
	return res, nil
}

var (
	flagColorMode = "auto"
	flagPortsRoot = "/usr/ports"
	flagVersion   bool

	flagOriginsSingleLine bool
	flagOriginOnly        bool
	flagSort              bool
	flagNoIndent          bool

	flagRegexp bool

	queries = queryFlags{
		{"d", grep.DEPENDS, ""},
		{"db", grep.BUILD_DEPENDS, ""},
		{"dl", grep.LIB_DEPENDS, ""},
		{"dr", grep.RUN_DEPENDS, ""},
		{"m", grep.MAINTAINER, ""},
		{"u", grep.USES, ""},
	}

	version = "devel"
)

func initFlags() {
	// disable GC, this is short-running utility and performance is more
	// important than memory consumpltion
	debug.SetGCPercent(-1)

	basename := path.Base(os.Args[0])

	if val, ok := os.LookupEnv("PORTSDIR"); ok {
		flagPortsRoot = val
	}

	flag.StringVar(&flagColorMode, "C", flagColorMode, "")
	flag.StringVar(&flagPortsRoot, "R", flagPortsRoot, "")
	flag.BoolVar(&flagVersion, "v", false, "")

	flag.BoolVar(&flagOriginsSingleLine, "1", false, "")
	flag.BoolVar(&flagOriginOnly, "o", false, "")
	flag.BoolVar(&flagSort, "s", false, "")
	flag.BoolVar(&flagNoIndent, "T", false, "")

	flag.BoolVar(&flagRegexp, "x", false, "")

	queries.addFlags()

	flag.Usage = func() {
		err := usageTemplate.Execute(os.Stderr, map[string]string{
			"basename":  basename,
			"portsRoot": flagPortsRoot,
			"colorMode": flagColorMode,
		})
		if err != nil {
			panic(err)
		}
	}

	flag.Parse()

	if flagPortsRoot == "" {
		fmt.Fprintln(os.Stderr, "ports tree root cannot be blank")
		flag.Usage()
		os.Exit(1)
	}

	if flagColorMode != colorModeAuto && flagColorMode != colorModeAlways && flagColorMode != colorModeNever {
		fmt.Fprintf(os.Stderr, "invalid color mode: %s\n", flagColorMode)
		flag.Usage()
		os.Exit(1)
	}

	if !queries.any() {
		flagOriginOnly = true
	}
}

func init() {
	initFlags()
	initFormatter()
}
