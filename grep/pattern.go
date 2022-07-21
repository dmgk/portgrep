package grep

import (
	"fmt"
	"regexp"
	"strings"
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
	Option() byte
	Description() string
	Compile(ctxBefore, ctxAfter int, quote bool) (*Regexp, error)

	optionString() string
	setQuery(query string)
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
	opt   byte
	pref  string
	desc  string
	pat   string
	query string
}

func (p *stringPattern) Option() byte {
	return p.opt
}

func (p *stringPattern) Description() string {
	if p.pref != "" {
		return fmt.Sprintf("-%c %s:query  %s", p.opt, p.pref, p.desc)
	}
	return fmt.Sprintf("-%c query    %s", p.opt, p.desc)
}

func (p *stringPattern) optionString() string {
	return string(p.opt) + ":"
}

func (p *stringPattern) setQuery(query string) {
	p.query = query
}

func (p *stringPattern) Compile(ctxBefore, ctxAfter int, quote bool) (*Regexp, error) {
	q := p.query
	if quote {
		q = regexp.QuoteMeta(q)
	}
	re, qsi, rsi, err := compile(fmt.Sprintf(p.pat, ctxBefore, q, ctxAfter))
	if err != nil {
		return nil, err
	}
	return &Regexp{re, qsi, rsi}, nil
}

type boolPattern struct {
	opt  byte
	pref string
	desc string
	pat  string
}

func (p *boolPattern) Option() byte {
	return p.opt
}

func (p *boolPattern) Description() string {
	if p.pref != "" {
		return fmt.Sprintf("-%c %s        %s", p.opt, p.pref, p.desc)
	}
	return fmt.Sprintf("-%c          %s", p.opt, p.desc)
}

func (p *boolPattern) setQuery(query string) {
	// noop
}

func (p *boolPattern) optionString() string {
	return string(p.opt)
}

func (p *boolPattern) Compile(ctxBefore, ctxAfter int, quote bool) (*Regexp, error) {
	re, qsi, rsi, err := compile(fmt.Sprintf(p.pat, ctxBefore, ctxAfter))
	if err != nil {
		return nil, err
	}
	return &Regexp{re, qsi, rsi}, nil
}

type Registry []Pattern

func (r Registry) OptionString() string {
	var b strings.Builder
	for _, p := range r {
		b.WriteString(p.optionString())
	}
	return b.String()
}

func (r Registry) Get(opt byte, query string) Pattern {
	for _, p := range r {
		if p.Option() == opt {
			p.setQuery(query)
			return p
		}
	}
	return nil
}

func Compile(query string, ctxBefore, ctxAfter int, quote bool) (*Regexp, error) {
	p := &stringPattern{
		// no query group, only result
		pat:   `(?:.*\n){0,%d}.*(?P<q>)(?P<r>%s).*(\n|\z)(?:.*\n){0,%d}`,
		query: query,
	}
	return p.Compile(ctxBefore, ctxAfter, quote)
}

var (
	portname = &stringPattern{
		opt:  'n',
		pref: "",
		desc: "search by PORTNAME",
		pat:  `(?i)(?:.*\n){0,%d}\b(?P<q>PORTNAME)\s*\??=\s*(?P<r>%s).*(\n|\z)(?:.*\n){0,%d}`,
	}
	maintainer = &stringPattern{
		opt:  'm',
		pref: "",
		desc: "search by MAINTAINER",
		pat:  `(?i)(?:.*\n){0,%d}\b(?P<q>MAINTAINER)\s*\??=\s*(?P<r>%s).*(\n|\z)(?:.*\n){0,%d}`,
	}
	buildDepends = &stringPattern{
		opt:  'b',
		pref: "",
		desc: "search by BUILD_DEPENDS",
		pat:  `(?:.*\n){0,%d}\b(?P<q>(\w+_)?BUILD_DEPENDS)\s*(\+|\?)?(=|=.*?[\s/}])(?P<r>%s)((\n|\z)|[\s@:>\.].*(\n|\z))(?:.*\n){0,%d}`,
	}
	libDepends = &stringPattern{
		opt:  'l',
		pref: "",
		desc: "search by LIB_DEPENDS",
		pat:  `(?:.*\n){0,%d}\b(?P<q>(\w+_)?LIB_DEPENDS)\s*(\+|\?)?(=|=.*?[\s/}])(?P<r>%s)((\n|\z)|[\s@:\.].*(\n|\z))(?:.*\n){0,%d}`,
	}
	runDepends = &stringPattern{
		opt:  'r',
		pref: "",
		desc: "search by RUN_DEPENDS",
		pat:  `(?:.*\n){0,%d}\b(?P<q>(\w+_)?RUN_DEPENDS)\s*(\+|\?)?(=|=.*?[\s/}])(?P<r>%s)((\n|\z)|[\s@:>\.].*(\n|\z))(?:.*\n){0,%d}`,
	}
	testDepends = &stringPattern{
		opt:  't',
		pref: "",
		desc: "search by TEST_DEPENDS",
		pat:  `(?:.*\n){0,%d}\b(?P<q>(\w+_)?TEST_DEPENDS)\s*(\+|\?)?(=|=.*?[\s/}])(?P<r>%s)((\n|\z)|[\s@:>\.].*(\n|\z))(?:.*\n){0,%d}`,
	}
	allDepends = &stringPattern{
		opt:  'd',
		pref: "",
		desc: "search by *_DEPENDS",
		pat:  `(?:.*\n){0,%d}\b(?P<q>(\w+_)?DEPENDS)\s*(\+|\?)?(=|=.*?[\s/}])(?P<r>%s)((\n|\z)|[\s@:>\.].*(\n|\z))(?:.*\n){0,%d}`,
	}
	onlyForArchs = &stringPattern{
		opt:  'a',
		pref: "",
		desc: "search by ONLY_FOR_ARCHS",
		pat:  `(?:.*\n){0,%d}\b(?P<q>ONLY_FOR_ARCHS)\s*(\+|\?)?(=|=.*?\s)(?P<r>%s)((\n|\z)|\s.*(\n|\z))(?:.*\n){0,%d}`,
	}
	uses = &stringPattern{
		opt:  'u',
		pref: "",
		desc: "search by USES",
		pat:  `(?:.*\n){0,%d}\b(?P<q>([\w_]+_)?USES)\s*(\+|\?)?(=|=.*?\s)(?P<r>%s)((\n|\z)|[\s:,].*(\n|\z))(?:.*\n){0,%d}`,
	}
	plist = &stringPattern{
		opt:  'p',
		pref: "",
		desc: "search by PLIST_FILES",
		pat:  `(?:.*\n){0,%d}\b(?P<q>([\w_]+_)?PLIST_FILES)\s*(\+|\?)?=.*?(?P<r>%s).*(\n|\z)(?:.*\n){0,%d}`,
	}
	broken = &boolPattern{
		opt:  'x',
		pref: "",
		desc: "search only ports marked BROKEN",
		pat:  `(?:.*\n){0,%d}\b(?P<q>BROKEN(_[^=]+)?)\s*\??=(?P<r>.*)(\n|\z)(?:.*\n){0,%d}`,
	}
)

var Patterns = Registry{
	portname,
	maintainer,
	buildDepends,
	libDepends,
	runDepends,
	testDepends,
	allDepends,
	onlyForArchs,
	uses,
	plist,
	broken,
}
