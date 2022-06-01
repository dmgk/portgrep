package grep

import (
	"flag"
	"fmt"
	"regexp"
)

type Regexp struct {
	re  *regexp.Regexp // compiled regexp
	qsi int            // query subexpression index
	rsi int            // result subexpression index
}

func (r *Regexp) Match(text []byte) (*Result, error) {
	smi := r.re.FindSubmatchIndex(text)
	if smi == nil {
		return nil, nil
	}
	if len(smi) <= r.rsi {
		return nil, fmt.Errorf("unexpected number of subexpressions %d in %v", len(smi), r)
	}
	res := &Result{
		Text: text[smi[0]:smi[1]],
	}
	if r.qsi >= 0 {
		res.QuerySubmatch = []int{smi[2*r.qsi] - smi[0], smi[2*r.qsi+1] - smi[0]}
	}
	if r.rsi >= 0 {
		res.ResultSubmatch = []int{smi[2*r.rsi] - smi[0], smi[2*r.rsi+1] - smi[0]}
	}
	return res, nil
}

type Pattern interface {
	Description() string
	Empty() bool
	Compile(ctxBefore, ctxAfter int) (*Regexp, error)
	CompileNoQuote(ctxBefore, ctxAfter int) (*Regexp, error)

	register()
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

	qsi, rsi := -1, -1
	for i, n := range re.SubexpNames() {
		if n == qsn {
			qsi = i
		}
		if n == rsn {
			rsi = i
		}
	}

	if qsi < 0 || rsi < 0 || rsi < qsi {
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
	return fmt.Sprintf("-%-2s query   %s", p.flag, p.desc)
}

func (p *stringPattern) Empty() bool {
	return p.val == ""
}

func (p *stringPattern) CompileNoQuote(ctxBefore, ctxAfter int) (*Regexp, error) {
	if p.Empty() {
		return nil, nil
	}

	re, qsi, rsi, err := compile(fmt.Sprintf(p.pat, ctxBefore, p.val, ctxAfter))
	if err != nil {
		return nil, err
	}

	return &Regexp{re, qsi, rsi}, nil
}

func (p *stringPattern) Compile(ctxBefore, ctxAfter int) (*Regexp, error) {
	p.val = regexp.QuoteMeta(p.val)
	return p.CompileNoQuote(ctxBefore, ctxAfter)
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
	return fmt.Sprintf("-%-2s         %s", p.flag, p.desc)
}

func (p *boolPattern) Empty() bool {
	return !p.val
}

func (p *boolPattern) CompileNoQuote(ctxBefore, ctxAfter int) (*Regexp, error) {
	if p.Empty() {
		return nil, nil
	}

	re, qsi, rsi, err := compile(fmt.Sprintf(p.pat, ctxBefore, ctxAfter))
	if err != nil {
		return nil, err
	}

	return &Regexp{re, qsi, rsi}, nil
}

func (p *boolPattern) Compile(ctxBefore, ctxAfter int) (*Regexp, error) {
	return p.CompileNoQuote(ctxBefore, ctxAfter)
}

func (p *boolPattern) register() {
	flag.BoolVar(&p.val, p.flag, false, p.desc)
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

// no query group, only result
const customPat = `(?:.*\n){0,%d}.*(?P<q>)(?P<r>%s).*(\n|\z)(?:.*\n){0,%d}`

func (s patternSlice) Compile(ctxBefore, ctxAfter int, valIsRegexp bool, custom ...string) ([]*Regexp, error) {
	// create patterns for custom queries
	var cps []Pattern
	for _, c := range custom {
		p := &stringPattern{
			pat: customPat,
			val: c,
		}
		cps = append(cps, p)
	}

	var res []*Regexp
	for _, p := range append(s, cps...) {
		var re *Regexp
		var err error
		if valIsRegexp {
			re, err = p.CompileNoQuote(ctxBefore, ctxAfter)
		} else {
			re, err = p.Compile(ctxBefore, ctxAfter)
		}
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

var (
	broken = &boolPattern{
		flag: "b",
		desc: "search only ports marked BROKEN",
		pat:  `(?:.*\n){0,%d}\b(?P<q>BROKEN(_[^=]+)?)\s*\??=(?P<r>.*)(\n|\z)(?:.*\n){0,%d}`,
	}
	depends = &stringPattern{
		flag: "d",
		desc: "search by *_DEPENDS",
		pat:  `(?:.*\n){0,%d}\b(?P<q>(\w+_)?DEPENDS)\s*(\+|\?)?(=|=.*?[\s/}])(?P<r>%s)((\n|\z)|[\s@:>\.].*(\n|\z))(?:.*\n){0,%d}`,
		val:  "",
	}
	buildDepends = &stringPattern{
		flag: "db",
		desc: "search by BUILD_DEPENDS",
		pat:  `(?:.*\n){0,%d}\b(?P<q>(\w+_)?BUILD_DEPENDS)\s*(\+|\?)?(=|=.*?[\s/}])(?P<r>%s)((\n|\z)|[\s@:>\.].*(\n|\z))(?:.*\n){0,%d}`,
		val:  "",
	}
	libDepends = &stringPattern{
		flag: "dl",
		desc: "search by LIB_DEPENDS",
		pat:  `(?:.*\n){0,%d}\b(?P<q>(\w+_)?LIB_DEPENDS)\s*(\+|\?)?(=|=.*?[\s/}])(?P<r>%s)((\n|\z)|[\s@:\.].*(\n|\z))(?:.*\n){0,%d}`,
		val:  "",
	}
	runDepends = &stringPattern{
		flag: "dr",
		desc: "search by RUN_DEPENDS",
		pat:  `(?:.*\n){0,%d}\b(?P<q>(\w+_)?RUN_DEPENDS)\s*(\+|\?)?(=|=.*?[\s/}])(?P<r>%s)((\n|\z)|[\s@:>\.].*(\n|\z))(?:.*\n){0,%d}`,
		val:  "",
	}
	testDepends = &stringPattern{
		flag: "dt",
		desc: "search by TEST_DEPENDS",
		pat:  `(?:.*\n){0,%d}\b(?P<q>(\w+_)?TEST_DEPENDS)\s*(\+|\?)?(=|=.*?[\s/}])(?P<r>%s)((\n|\z)|[\s@:>\.].*(\n|\z))(?:.*\n){0,%d}`,
		val:  "",
	}
	onlyForArchs = &stringPattern{
		flag: "oa",
		desc: "search by ONLY_FOR_ARCHS",
		pat:  `(?:.*\n){0,%d}\b(?P<q>ONLY_FOR_ARCHS)\s*(\+|\?)?(=|=.*?\s)(?P<r>%s)((\n|\z)|\s.*(\n|\z))(?:.*\n){0,%d}`,
		val:  "",
	}
	maintainer = &stringPattern{
		flag: "m",
		desc: "search by MAINTAINER",
		pat:  `(?i)(?:.*\n){0,%d}\b(?P<q>MAINTAINER)\s*\??=\s*(?P<r>%s).*(\n|\z)(?:.*\n){0,%d}`,
		val:  "",
	}
	portname = &stringPattern{
		flag: "n",
		desc: "search by PORTNAME",
		pat:  `(?i)(?:.*\n){0,%d}\b(?P<q>PORTNAME)\s*\??=\s*(?P<r>%s).*(\n|\z)(?:.*\n){0,%d}`,
		val:  "",
	}
	uses = &stringPattern{
		flag: "u",
		desc: "search by USES",
		pat:  `(?:.*\n){0,%d}\b(?P<q>([\w_]+_)?USES)\s*(\+|\?)?(=|=.*?\s)(?P<r>%s)((\n|\z)|[\s:,].*(\n|\z))(?:.*\n){0,%d}`,
		val:  "",
	}
	plist = &stringPattern{
		flag: "pl",
		desc: "search by PLIST_FILES",
		pat:  `(?:.*\n){0,%d}\b(?P<q>([\w_]+_)?PLIST_FILES)\s*(\+|\?)?=.*?(?P<r>%s).*(\n|\z)(?:.*\n){0,%d}`,
		val:  "",
	}
)

var Patterns = patternSlice{
	broken,
	depends,
	buildDepends,
	libDepends,
	runDepends,
	testDepends,
	maintainer,
	portname,
	onlyForArchs,
	plist,
	uses,
}

func init() {
	Patterns.register()
}
