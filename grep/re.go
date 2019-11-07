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
	BUILD_DEPENDS: `(?m)\b(BUILD_DEPENDS)=(?:.*\s)?(?:.+/)?(%s)(?:$|[\s:>].*$)`,
	LIB_DEPENDS:   `(?m)\b(LIB_DEPENDS)=(?:.*\s)?(?:.+/)?(%s)(?:$|[\s:].*$)`,
	RUN_DEPENDS:   `(?m)\b(RUN_DEPENDS)=(?:.*\s)?(?:.+/)?(%s)(?:$|[\s:>].*$)`,
	DEPENDS:       `(?m)\b((?:[\w_]+_)?DEPENDS)=(?:.*\s)?(?:.+/)?(%s)(?:$|[\s:>].*$)`,
	MAINTAINER:    `(?i)\b(MAINTAINER)=\s*(%s).*`,
	USES:          `(?m)\b((?:[\w_]+_)?USES)=(?:.*\s)?(%s)(?:$|[\s:].*$)`,
}
