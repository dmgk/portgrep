package grep

import (
	"fmt"
	"regexp"
)

const (
	MAINTAINER = iota + 1
	USES
)

func Compile(id int, s string) (*regexp.Regexp, error) {
	f, ok := re[id]
	if !ok {
		return nil, fmt.Errorf("unknown regexp id: %d", id)
	}
	return regexp.Compile(fmt.Sprintf(f, regexp.QuoteMeta(s)))
}

var re = map[int]string{
	MAINTAINER: `(?i)\b(MAINTAINER)=\s*(%s).*`,
	USES:       `(?m)(?:\b|_)(USES)=(?:.*\s)?(%s)(?:$|(?:\s|:).*$)`,
}
