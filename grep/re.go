package grep

import (
	"fmt"
	"regexp"
)

const (
	BUILD_DEPENDS = iota + 1
	LIB_DEPENDS
	RUN_DEPENDS
	DEPENDS
	MAINTAINER
	USES
)

type Regexp struct {
	re  *regexp.Regexp // compiled regexp
	qsi int            // query subexpression index
	rsi int            // result subexpression index
}

func Compile(kind int, s string, x bool) (*Regexp, error) {
	f, ok := re[kind]
	if !ok {
		panic(fmt.Sprintf("unknown regexp kind: %d", kind))
	}

	if !x {
		s = regexp.QuoteMeta(s)
	}

	re, err := regexp.Compile(fmt.Sprintf(f, s))
	if err != nil {
		return nil, err
	}

	res := &Regexp{re: re, qsi: -1, rsi: -1}

	for i, n := range re.SubexpNames() {
		if n == qsn {
			res.qsi = i
		}
		if n == rsn {
			res.rsi = i
		}
	}

	if res.qsi < 0 || res.rsi < 0 {
		panic(fmt.Sprintf("invalid query/result subexpressions: %s", re))
	}

	return res, nil
}

const (
	qsn = "q" // query subexpression name
	rsn = "r" // result subexpression name
)

var re = map[int]string{
	BUILD_DEPENDS: `\b(?P<q>BUILD_DEPENDS)\s*(=|=.*?[\s/}])(?P<r>%s)[\s@:>\.].*(\n|\z)`,
	LIB_DEPENDS:   `\b(?P<q>LIB_DEPENDS)\s*(=|=.*?[\s/}])(?P<r>%s)[\s@:\.].*(\n|\z)`,
	RUN_DEPENDS:   `\b(?P<q>RUN_DEPENDS)\s*(=|=.*?[\s/}])(?P<r>%s)[\s@:>\.].*(\n|\z)`,
	DEPENDS:       `\b(?P<q>(\w+_)?DEPENDS)\s*(=|=.*?[\s/}])(?P<r>%s)[\s@:>\.].*(\n|\z)`,
	MAINTAINER:    `(?i)\b(?P<q>MAINTAINER)\s*=\s*(?P<r>%s).*(\n|\z)`,
	USES:          `\b(?P<q>([\w_]+_)?USES)\s*(=|=.*?[\s])(?P<r>%s)((\n|\z)|[\s:,].*(\n|\z))`,
}
