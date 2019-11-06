package main

import (
	"flag"
	"fmt"
	"html/template"
	"os"
	"path"
	"regexp"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/dmgk/portgrep/grep"
)

func main() {
	flag.Parse()

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
	rxs, err := regexps()
	if err != nil {
		return err
	}

	prefix := flagPortsRoot + "/"

	type r struct {
		origin  string
		matches [][][]byte
	}
	var rr []*r

	fn := func(path string, matches [][][]byte, err error) error {
		if err != nil {
			return err
		}

		rr = append(rr, &r{
			origin:  strings.TrimPrefix(path, prefix),
			matches: matches,
		})

		return nil
	}

	err = grep.Grep(flagPortsRoot, rxs, fn, runtime.NumCPU())
	if err != nil {
		return err
	}

	sort.Slice(rr, func(i, j int) bool {
		return rr[i].origin < rr[j].origin
	})

	var needSep bool

	for _, r := range rr {
		if flagOneLine {
			if needSep {
				fmt.Print(" ")
			}
			fmt.Print(r.origin)
			needSep = true
			continue
		}

		if flagOriginOnly {
			fmt.Println(r.origin)
			continue
		}

		if r.matches != nil {
			fmt.Printf("%s:\n", r.origin)
			for _, m := range r.matches {
				fmt.Printf("\t%s\n", string(m[0]))
			}
		}
	}

	return nil
}

func runUnsorted() error {
	rxs, err := regexps()
	if err != nil {
		return err
	}

	prefix := flagPortsRoot + "/"
	var needSep bool

	fn := func(path string, matches [][][]byte, err error) error {
		if err != nil {
			return err
		}

		if flagOneLine {
			if needSep {
				fmt.Print(" ")
			}
			fmt.Print(strings.TrimPrefix(path, prefix))
			needSep = true
			return nil
		}

		if flagOriginOnly {
			fmt.Println(strings.TrimPrefix(path, prefix))
			return nil
		}

		if matches != nil {
			fmt.Printf("%s:\n", strings.TrimPrefix(path, prefix))
			for _, m := range matches {
				fmt.Printf("\t%s\n", string(m[0]))
			}
		}

		return nil
	}

	return grep.Grep(flagPortsRoot, rxs, fn, runtime.NumCPU())
}

func regexps() ([]*regexp.Regexp, error) {
	var res []*regexp.Regexp

	if queryMaintainer != "" {
		re, err := grep.Compile(grep.MAINTAINER, queryMaintainer, flagRegexp)
		if err != nil {
			return nil, err
		}
		res = append(res, re)
	}

	if queryUses != "" {
		re, err := grep.Compile(grep.USES, queryUses, flagRegexp)
		if err != nil {
			return nil, err
		}
		res = append(res, re)
	}

	return res, nil
}

var (
	flagPortsRoot   = "/usr/ports"
	flagVersion     bool
	flagOneLine     bool
	flagOriginOnly  bool
	flagSort        bool
	flagRegexp      bool
	queryMaintainer string
	queryUses       string

	version = "devel"
)

func init() {
	// disable GC, this is short-running utility and performance is more
	// important than memory consumpltion
	debug.SetGCPercent(-1)

	basename := path.Base(os.Args[0])

	if val, ok := os.LookupEnv("PORTSDIR"); ok {
		flagPortsRoot = val
	}

	flag.StringVar(&flagPortsRoot, "R", flagPortsRoot, "")
	flag.BoolVar(&flagVersion, "v", false, "")

	flag.BoolVar(&flagOneLine, "1", false, "")
	flag.BoolVar(&flagOriginOnly, "o", false, "")
	flag.BoolVar(&flagSort, "s", false, "")
	flag.BoolVar(&flagRegexp, "x", false, "")

	flag.StringVar(&queryMaintainer, "m", "", "")
	flag.StringVar(&queryUses, "u", "", "")

	flag.Usage = func() {
		err := usageTemplate.Execute(os.Stderr, map[string]string{
			"basename":  basename,
			"portsRoot": flagPortsRoot,
		})
		if err != nil {
			panic(err)
		}
	}
}

var usageTemplate = template.Must(template.New("Usage").Parse(`Usage: {{.basename}} <options>

Global options:
  -R path   ports tree root (default: {{.portsRoot}})
  -v        show version

Search options:
  -1        output origins in a single line (implies -o)
  -o        output origins only
  -s        sort results by origin
  -x        treat query as a regular expression
  -m query  search by MAINTAINER
  -u query  search by USES
`))
