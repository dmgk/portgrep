package main

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"unicode"

	"github.com/dmgk/getopt"
	"github.com/dmgk/portgrep/formatter"
	"github.com/dmgk/portgrep/grep"
	"github.com/mattn/go-isatty"
)

var usageTmpl = template.Must(template.New("usage").Parse(`
usage: {{.progname}} [options] [query ...]

General options:
  -h          show help and exit
  -V          show version and exit
  -R path     ports tree root (default: {{.portsRoot}})
  -M mode     colorized output mode: [auto|never|always] (default: {{.colorMode}})
  -G colors   set colors (default: "{{.colors}}")
              the order is query,match,path,separator; see ls(1) for color codes

Search options:
  -c name,... limit search to only these categories
  -O          multiple searches are OR-ed (default: AND-ed)
  -F          interpret query as a plain text, not regular expression
  -j jobs     number of parallel jobs (default: {{.maxJobs}})

Formatting options:
  -1          output origins in a single line (implies -o)
  -A count    show count lines of context after match
  -B count    show count lines of context before match
  -C count    show count lines of context around match
  -o          output origins only
  -s          sort results by origin
  -T          do not indent results

Predefined searches:{{range .patterns}}
  {{.Description}}{{end}}
`[1:]))

var (
	progname          string
	version           = "devel"
	portsRoot         = "/usr/ports"
	colorMode         = "auto"
	colors            = formatter.DefaultColors
	categories        []string
	ored              bool
	plainText         bool
	maxJobs           = runtime.NumCPU()
	originsSingleLine bool
	contextAfter      int
	contextBefore     int
	originsOnly       bool
	noIndent          bool
)

const (
	colorModeAuto   = "auto"
	colorModeAlways = "always"
	colorModeNever  = "never"
)

func showUsage() {
	err := usageTmpl.Execute(os.Stdout, map[string]interface{}{
		"progname":  progname,
		"colorMode": colorMode,
		"colors":    colors,
		"maxJobs":   maxJobs,
		"patterns":  grep.Patterns,
	})
	if err != nil {
		panic(fmt.Sprintf("error executing template %s: %v", usageTmpl.Name(), err))
	}
}

func showVersion() {
	fmt.Printf("%s %s\n", progname, version)
}

func errExit(format string, v ...interface{}) {
	fmt.Fprint(os.Stderr, progname, ": ")
	fmt.Fprintf(os.Stderr, format, v...)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

func main() {
	// disable GC, this is short-running utility and performance is more
	// important than memory consumption
	debug.SetGCPercent(-1)

	if v, ok := os.LookupEnv("PORTSDIR"); ok && v != "" {
		portsRoot = v
	}
	if v, ok := os.LookupEnv("PORTGREP_COLORS"); ok && v != "" {
		colors = v
	}

	opts, err := getopt.NewArgv("hVR:M:G:c:OFj:1A:B:C:osT"+grep.Patterns.OptionString(), argsWithDefaults(os.Args, "PORTGREP_OPTS"))
	if err != nil {
		panic(fmt.Sprintf("error creating options parser: %s", err))
	}
	progname = opts.ProgramName()

	var pts []grep.Pattern

	for opts.Scan() {
		opt, err := opts.Option()
		if err != nil {
			errExit(err.Error())
		}

		switch opt.Opt {
		case 'h':
			showUsage()
			os.Exit(0)
		case 'V':
			showVersion()
			os.Exit(0)
		case 'R':
			portsRoot = opt.String()
		case 'M':
			switch opt.String() {
			case colorModeAuto, colorModeNever, colorModeAlways:
				colorMode = opt.String()
			default:
				errExit("-M: invalid color mode: %s", opt.String())
			}
		case 'G':
			colors = opt.String()
		case 'c':
			categories = splitOptions(opt.String())
		case 'O':
			ored = true
		case 'F':
			plainText = true
		case 'j':
			v, err := opt.Int()
			if err != nil {
				errExit("-j: %s", err.Error())
			}
			if v <= 0 {
				v = 1
			}
			maxJobs = v
		case '1':
			originsSingleLine = true
		case 'A':
			v, err := opt.Int()
			if err != nil {
				errExit("-A: %s", err)
			}
			contextAfter = v
		case 'B':
			v, err := opt.Int()
			if err != nil {
				errExit("-B: %s", err)
			}
			contextBefore = v
		case 'C':
			v, err := opt.Int()
			if err != nil {
				errExit("-C: %s", err)
			}
			contextBefore = v
			contextAfter = v
		case 'o':
			originsOnly = true
		case 's':
			maxJobs = 1
		case 'T':
			noIndent = true
		default:
			p := grep.Patterns.Get(opt.Opt, opt.String())
			if p == nil {
				panic("unhandled option: -" + string(opt.Opt))
			}
			pts = append(pts, p)
		}
	}

	var rxs []*grep.Regexp

	for _, p := range pts {
		rx, err := p.Compile(contextBefore, contextAfter, plainText)
		if err != nil {
			errExit("-%c: %s", p.Option(), err)
		}
		rxs = append(rxs, rx)
	}
	for _, q := range opts.Args() {
		rx, err := grep.Compile(q, contextBefore, contextAfter, plainText)
		if err != nil {
			errExit("query %q: %s", q, err)
		}
		rxs = append(rxs, rx)
	}

	// show only origins if neither predefined query or custom regexp provided
	if len(rxs) == 0 {
		originsOnly = true
	}

	f := initFormatter()
	gfn := func(path string, results grep.Results, err error) error {
		if err != nil {
			return err
		}
		return f.Format(path, results)
	}
	if err := grep.Grep(portsRoot, categories, rxs, ored, gfn, maxJobs); err != nil {
		errExit(err.Error())
	}
}

func initFormatter() formatter.Formatter {
	var w io.Writer = os.Stdout
	flags := formatter.Fdefaults
	term := isatty.IsTerminal(os.Stdout.Fd())

	if colorMode == colorModeAlways || (term && colorMode == colorModeAuto) {
		flags |= formatter.Fcolor
		if colors != "" {
			formatter.SetColors(colors)
		}
	}
	if originsSingleLine {
		flags |= formatter.ForiginsSingleLine
	}
	if originsOnly {
		flags |= formatter.ForiginsOnly
	}

	f := formatter.NewText(w, portsRoot, flags)
	if !noIndent {
		f.SetIndent("\t")
	}
	return f
}

func argsWithDefaults(argv []string, env string) []string {
	args := argv[1:]
	if v, ok := os.LookupEnv(env); ok && v != "" {
		args = append(splitOptions(v), args...)
	}
	return append([]string{argv[0]}, args...)
}

func splitOptions(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || r == ','
	})
}
