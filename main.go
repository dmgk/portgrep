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
	"strings"

	"github.com/dmgk/portgrep/formatter"
	"github.com/dmgk/portgrep/grep"
	"github.com/mattn/go-isatty"
)

func main() {
	if flag.NFlag() == 0 && flag.NArg() == 0 {
		flag.Usage()
		os.Exit(0)
	}

	if flagVersion {
		fmt.Fprintln(os.Stderr, version)
		os.Exit(0)
	}

	var err error

	if flagSort {
		err = runSorted(flag.Args()...)
	} else {
		err = runUnsorted(flag.Args()...)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(0)
	}
}

func runUnsorted(custom ...string) error {
	rxs, err := grep.Patterns.Compile(flagRegexp, custom...)
	if err != nil {
		return err
	}

	fn := func(path string, results grep.Results, err error) error {
		if err != nil {
			return err
		}
		return form.Format(path, results)
	}

	var cats []string
	if flagCategories != "" {
		cats = strings.Split(flagCategories, ",")
	}

	return grep.Grep(flagPortsRoot, cats, rxs, flagOred, fn, runtime.NumCPU())
}

func runSorted(custom ...string) error {
	rxs, err := grep.Patterns.Compile(flagRegexp, custom...)
	if err != nil {
		return err
	}

	type r struct {
		path    string
		results grep.Results
	}
	var rr []*r

	fn := func(path string, results grep.Results, err error) error {
		if err != nil {
			return err
		}
		rr = append(rr, &r{path, results})
		return nil
	}

	var cats []string
	if flagCategories != "" {
		cats = strings.Split(flagCategories, ",")
	}

	err = grep.Grep(flagPortsRoot, cats, rxs, flagOred, fn, runtime.NumCPU())
	if err != nil {
		return err
	}

	sort.Slice(rr, func(i, j int) bool {
		return rr[i].path < rr[j].path
	})

	for _, r := range rr {
		if err := form.Format(r.path, r.results); err != nil {
			return err
		}
	}

	return nil
}

var (
	flagColorMode = "auto"
	flagPortsRoot = "/usr/ports"
	flagVersion   bool

	flagCategories string
	flagOred       bool
	flagRegexp     bool

	flagOriginsSingleLine bool
	flagOriginOnly        bool
	flagSort              bool
	flagNoIndent          bool
)

var version = "devel"

var usageTemplate = template.Must(template.New("Usage").Parse(`Usage: {{.basename}} [options] [regexp ...]

General options:
  -C mode     colorized output mode: [auto|never|always] (default: {{.colorMode}})
  -R path     ports tree root (default: {{.portsRoot}})
  -v          show version and exit

Search options:
  -c cat,...  limit search to only these categories
  -O          multiple searches are OR-ed (default: AND-ed)
  -x          treat query as a regular expression

Formatting options:
  -1          output origins in a single line (implies -o)
  -o          output origins only
  -s          sort results by origin
  -T          do not indent results

Predefined searches:{{range .patterns}}
  {{.Description}}{{end}}
`))

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

	flag.StringVar(&flagCategories, "c", flagCategories, "")
	flag.BoolVar(&flagOred, "O", false, "")
	flag.BoolVar(&flagRegexp, "x", false, "")

	flag.BoolVar(&flagOriginsSingleLine, "1", false, "")
	flag.BoolVar(&flagOriginOnly, "o", false, "")
	flag.BoolVar(&flagSort, "s", false, "")
	flag.BoolVar(&flagNoIndent, "T", false, "")

	flag.Usage = func() {
		err := usageTemplate.Execute(os.Stderr, map[string]interface{}{
			"basename":  basename,
			"portsRoot": flagPortsRoot,
			"colorMode": flagColorMode,
			"patterns":  grep.Patterns,
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

	// neither predefined query or custom regexp provided
	if grep.Patterns.Empty() && flag.NArg() == 0 {
		flagOriginOnly = true
	}
}

const (
	colorModeAuto   = "auto"
	colorModeAlways = "always"
	colorModeNever  = "never"
)

var form formatter.Formatter

func initFormatter() {
	var w io.Writer = os.Stdout
	flags := formatter.Fdefaults
	term := isatty.IsTerminal(os.Stdout.Fd())

	if flagColorMode == colorModeAlways || (term && flagColorMode == colorModeAuto) {
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

func init() {
	initFlags()
	initFormatter()
}
