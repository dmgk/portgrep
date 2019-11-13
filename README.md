`portgrep` is a fast parallel ports tree search utility.

[![Build Status](https://travis-ci.org/dmgk/portgrep.svg?branch=master)](https://travis-ci.org/dmgk/portgrep)

#### Installation

    go get github.com/dmgk/portgrep

#### Usage

```
Usage: portgrep <options>

Global options:
  -C mode    colorized output mode: [auto|never|always] (default: auto)
  -R path    ports tree root (default: /usr/ports)
  -v         show version

Formatting options:
  -1         output origins in a single line (implies -o)
  -o         output origins only
  -s         sort results by origin
  -T         do not indent results

Search options:
  -x         treat query as a regular expression
  -b         search only ports marked BROKEN
  -d  query  search by *_DEPENDS
  -db query  search by BUILD_DEPENDS
  -dl query  search by LIB_DEPENDS
  -dr query  search by RUN_DEPENDS
  -oa query  search by ONLY_FOR_ARCHS
  -m  query  search by MAINTAINER
  -n  query  search by PORTNAME
  -u  query  search by USES
```

Example: find all broken Go ports:

```sh
$ portgrep -u go -b
databases/cayley:
        BROKEN_i386= gopkg.in/mgo.v2/bson/json.go:320:7: constant 9007199254740992 overflows int
        USES=  go:modules
databases/mongodb34-tools:
        BROKEN_SSL= openssl111 libressl libressl-devel
        USES= go localbase
databases/mongodb36-tools:
        BROKEN_SSL= openssl111 libressl libressl-devel
        USES= go localbase
misc/exercism:
        BROKEN=  unfetchable
        USES=  go
devel/grumpy:
        BROKEN_i386= constant 2147762812 overflows int
        USES=  gmake go:no_targets,run python:2.7 shebangfix
...
 ```

#### Performance

```shell
$ time (find /usr/ports -name Makefile | xargs grep "MAINTAINER=.*ports@" >/dev/null)

real    0m1.045s
user    0m0.237s
sys     0m1.076s
```

```shell
$ time (./portgrep -m ports@ >/dev/null)

real    0m0.395s
user    0m2.238s
sys     0m0.796s
```
