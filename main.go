package main

import (
	"flag"
	"fmt"
	"html/template"
	"os"
	"path"
	"regexp"
	"runtime"
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

	fn := func(path string, matches [][][]byte, err error) bool {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return false
		}

		rr = append(rr, &r{
			origin:  strings.TrimPrefix(path, prefix),
			matches: matches,
		})

		return true
	}

	err = grep.Grep(flagPortsRoot, rxs, fn, runtime.NumCPU())
	if err != nil {
		return err
	}

	sort.Slice(rr, func(i, j int) bool {
		return rr[i].origin < rr[j].origin
	})

	for _, r := range rr {
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

	fn := func(path string, matches [][][]byte, err error) bool {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return false
		}

		if flagOriginOnly {
			fmt.Println(strings.TrimPrefix(path, prefix))
			return true
		}

		if matches != nil {
			fmt.Printf("%s:\n", strings.TrimPrefix(path, prefix))
			for _, m := range matches {
				fmt.Printf("\t%s\n", string(m[0]))
			}
		}

		return true
	}

	return grep.Grep(flagPortsRoot, rxs, fn, runtime.NumCPU())
}

func regexps() ([]*regexp.Regexp, error) {
	var res []*regexp.Regexp

	if searchMaintainer != "" {
		re, err := grep.Compile(grep.MAINTAINER, searchMaintainer)
		if err != nil {
			return nil, err
		}
		res = append(res, re)
	}

	if searchUses != "" {
		re, err := grep.Compile(grep.USES, searchUses)
		if err != nil {
			return nil, err
		}
		res = append(res, re)
	}

	return res, nil
}

var (
	flagPortsRoot    = "/usr/ports"
	flagOriginOnly   bool
	flagSort         bool
	searchMaintainer string
	searchUses       string
)

var (
	flagVersion bool
	version     = "devel"
)

func init() {
	basename := path.Base(os.Args[0])

	if val, ok := os.LookupEnv("PORTS_ROOT"); ok {
		flagPortsRoot = val
	}

	flag.StringVar(&flagPortsRoot, "R", flagPortsRoot, "ports tree root")
	flag.BoolVar(&flagOriginOnly, "o", false, "output origins only")
	flag.BoolVar(&flagSort, "s", false, "sort results by origin")
	flag.BoolVar(&flagVersion, "v", false, "show version")
	flag.StringVar(&searchMaintainer, "m", "", "search by maintainer")
	flag.StringVar(&searchUses, "u", "", "search by USES")

	flag.Usage = func() {
		usageTemplate.Execute(os.Stderr, map[string]string{
			"basename":  basename,
			"portsRoot": flagPortsRoot,
		})
	}
}

var usageTemplate = template.Must(template.New("Usage").Parse(`usage: {{.basename}} <options>

Global options:
  -R ROOT        ports tree root (default: {{.portsRoot}})
  -v             show version

Search options:
  -o             output origins only
  -s             sort results by origin
  -m MAINTAINER  search by MAINTAINER
  -u USES        search by USES
`))
