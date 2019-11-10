package grep

import (
	"flag"
	"fmt"
	"regexp"
)

type Pattern interface {
	Description() string
	Empty() bool
	Compile(isRegexp bool) (Regexp, error)

	register()
}

type patternSlice []Pattern

func (s patternSlice) Empty() bool {
	for _, p := range s {
		if !p.Empty() {
			return false
		}
	}
	return true
}

func (s patternSlice) Compile(isRegexp bool) ([]Regexp, error) {
	var res []Regexp
	for _, p := range s {
		re, err := p.Compile(isRegexp)
		if err != nil {
			return nil, err
		}
		if re != nil {
			res = append(res, re)
		}
	}
	return res, nil
}

func (s patternSlice) register() {
	for _, p := range s {
		p.register()
	}
}

const (
	qsn = "q" // query subexpression name
	rsn = "r" // result subexpression name
)

func compile(pat string) (*regexp.Regexp, int, int, error) {
	re, err := regexp.Compile(pat)
	if err != nil {
		return nil, 0, 0, err
	}

	var qsi, rsi int
	for i, n := range re.SubexpNames() {
		if n == qsn {
			qsi = i
		}
		if n == rsn {
			rsi = i
		}
	}

	if qsi < 0 || rsi < 0 {
		return nil, 0, 0, fmt.Errorf("invalid subexpressions: %s", re)
	}

	return re, qsi, rsi, nil
}

type stringPattern struct {
	flag string
	desc string
	pat  string
	val  string
}

func (p *stringPattern) Description() string {
	return fmt.Sprintf("-%-2s query  %s", p.flag, p.desc)
}

func (p *stringPattern) Empty() bool {
	return p.val == ""
}

func (p *stringPattern) Compile(isRegexp bool) (Regexp, error) {
	if p.Empty() {
		return nil, nil
	}

	q := p.val
	if !isRegexp {
		q = regexp.QuoteMeta(q)
	}

	re, qsi, rsi, err := compile(fmt.Sprintf(p.pat, q))
	if err != nil {
		return nil, err
	}

	return &stringRegexp{re, qsi, rsi}, nil
}

func (p *stringPattern) register() {
	flag.StringVar(&p.val, p.flag, "", p.desc)
}

type boolPattern struct {
	flag string
	desc string
	pat  string
	val  bool
}

func (p *boolPattern) Description() string {
	return fmt.Sprintf("-%-2s        %s", p.flag, p.desc)
}

func (p *boolPattern) Empty() bool {
	return !p.val
}

func (p *boolPattern) Compile(isRegexp bool) (Regexp, error) {
	if p.Empty() {
		return nil, nil
	}

	re, qsi, rsi, err := compile(p.pat)
	if err != nil {
		return nil, err
	}

	return &boolRegexp{re, qsi, rsi}, nil
}

func (p *boolPattern) register() {
	flag.BoolVar(&p.val, p.flag, false, p.desc)
}

type Regexp interface {
	Match(text []byte) (*Result, error)
}

type stringRegexp struct {
	re  *regexp.Regexp // compiled regexp
	qsi int            // query subexpression index
	rsi int            // result subexpression index
}

func (r *stringRegexp) Match(text []byte) (*Result, error) {
	smi := r.re.FindSubmatchIndex(text)
	if smi == nil {
		return nil, nil
	}
	if len(smi) <= r.rsi {
		return nil, fmt.Errorf("unexpected number of subexpressions %d in %v", len(smi), r)
	}
	return &Result{
		Text:           text[smi[0]:smi[1]],
		QuerySubmatch:  []int{smi[2*r.qsi] - smi[0], smi[2*r.qsi+1] - smi[0]},
		ResultSubmatch: []int{smi[2*r.rsi] - smi[0], smi[2*r.rsi+1] - smi[0]},
	}, nil
}

type boolRegexp struct {
	re  *regexp.Regexp // compiled regexp
	qsi int            // query subexpression index
	rsi int            // result subexpression index
}

func (r *boolRegexp) Match(text []byte) (*Result, error) {
	smi := r.re.FindSubmatchIndex(text)
	if smi == nil {
		return nil, nil
	}
	if len(smi) <= r.rsi {
		return nil, fmt.Errorf("unexpected number of subexpressions %d in %v", len(smi), r)
	}
	return &Result{
		Text:           text[smi[0]:smi[1]],
		QuerySubmatch:  []int{smi[2*r.qsi] - smi[0], smi[2*r.qsi+1] - smi[0]},
		ResultSubmatch: []int{smi[2*r.rsi] - smi[0], smi[2*r.rsi+1] - smi[0]},
	}, nil
}

var (
	broken = &boolPattern{
		flag: "b",
		desc: "search only ports marked BROKEN/IGNORE",
		pat:  `\b(?P<q>BROKEN(_[^=])?)\s*=(?P<r>.*)(\n|\z)`,
	}
	depends = &stringPattern{
		flag: "d",
		desc: "search by *_DEPENDS",
		pat:  `\b(?P<q>(\w+_)?DEPENDS)\s*(=|=.*?[\s/}])(?P<r>%s)[\s@:>\.].*(\n|\z)`,
		val:  "",
	}
	buildDepends = &stringPattern{
		flag: "db",
		desc: "search by BUILD_DEPENDS",
		pat:  `\b(?P<q>BUILD_DEPENDS)\s*(=|=.*?[\s/}])(?P<r>%s)[\s@:>\.].*(\n|\z)`,
		val:  "",
	}
	libDepends = &stringPattern{
		flag: "dl",
		desc: "search by LIB_DEPENDS",
		pat:  `\b(?P<q>LIB_DEPENDS)\s*(=|=.*?[\s/}])(?P<r>%s)[\s@:\.].*(\n|\z)`,
		val:  "",
	}
	runDepends = &stringPattern{
		flag: "dr",
		desc: "search by RUN_DEPENDS",
		pat:  `\b(?P<q>RUN_DEPENDS)\s*(=|=.*?[\s/}])(?P<r>%s)[\s@:>\.].*(\n|\z)`,
		val:  "",
	}
	onlyForArchs = &stringPattern{
		flag: "oa",
		desc: "search by ONLY_FOR_ARCHS",
		pat:  `\b(?P<q>ONLY_FOR_ARCHS)\s*(=|=.*?\s)(?P<r>%s)((\n|\z)|\s.*(\n|\z))`,
		val:  "",
	}
	maintainer = &stringPattern{
		flag: "m",
		desc: "search by MAINTAINER",
		pat:  `(?i)\b(?P<q>MAINTAINER)\s*=\s*(?P<r>%s).*(\n|\z)`,
		val:  "",
	}
	uses = &stringPattern{
		flag: "u",
		desc: "search by USES",
		pat:  `\b(?P<q>([\w_]+_)?USES)\s*(=|=.*?\s)(?P<r>%s)((\n|\z)|[\s:,].*(\n|\z))`,
		val:  "",
	}
)

var Patterns = patternSlice{
	broken,
	depends,
	buildDepends,
	libDepends,
	runDepends,
	onlyForArchs,
	maintainer,
	uses,
}

func init() {
	Patterns.register()
}