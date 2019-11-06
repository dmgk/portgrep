package grep

import (
	"fmt"
	"regexp"
)

const (
	MAINTAINER = iota + 1
	USES
)

func Compile(kind int, s string, sre bool) (*regexp.Regexp, error) {
	f, ok := re[kind]
	if !ok {
		return nil, fmt.Errorf("unknown regexp kind: %d", kind)
	}
	if sre {
		return regexp.Compile(fmt.Sprintf(f, s))
	}
	return regexp.Compile(fmt.Sprintf(f, regexp.QuoteMeta(s)))
}

var re = map[int]string{
	MAINTAINER: `(?i)\b(MAINTAINER)=\s*(%s).*`,
	USES:       `(?m)(?:\b|_)(USES)=(?:.*\s)?(%s)(?:$|(?:\s|:).*$)`,
}
