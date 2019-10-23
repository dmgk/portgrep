`portgrep` is a fast parallel port search utility.


#### Usage

```
usage: portgrep <options>

Global options:
  -R ROOT        ports tree root (default: /usr/ports)
  -v             show version

Search options:
  -o             output origins only
  -s             sort by origin
  -m MAINTAINER  search by MAINTAINER
  -u USES        search by USES
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
