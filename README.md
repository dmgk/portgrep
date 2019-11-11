`portgrep` is a fast parallel port search utility.

[![Build Status](https://travis-ci.org/dmgk/portgrep.svg?branch=master)](https://travis-ci.org/dmgk/portgrep)

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
  -u  query  search by USES
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
