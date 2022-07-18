## portgrep

portgrep is a fast parallel ports tree search utility.

![Tests](https://github.com/dmgk/portgrep/actions/workflows/tests.yml/badge.svg)

#### Installation

    go install github.com/dmgk/portgrep@latest

#### Usage

```
usage: portgrep [options] [query ...]

General options:
  -h          show help and exit
  -V          show version and exit
  -R path     ports tree root (default: )
  -M mode     colorized output mode: [auto|never|always] (default: auto)
  -G colors   set colors (default: "BCDA")
              the order is query,match,path,separator; see ls(1) for color codes

Search options:
  -c name,... limit search to only these categories
  -O          multiple searches are OR-ed (default: AND-ed)
  -F          interpret query as a plain text, not regular expression
  -j jobs     number of parallel jobs (default: 8)

Formatting options:
  -1          output origins in a single line (implies -o)
  -A count    show count lines of context after match
  -B count    show count lines of context before match
  -C count    show count lines of context around match
  -o          output origins only
  -s          sort results by origin
  -T          do not indent results

Predefined searches:
  -d query    search by *_DEPENDS
  -b query    search by BUILD_DEPENDS
  -l query    search by LIB_DEPENDS
  -r query    search by RUN_DEPENDS
  -t query    search by TEST_DEPENDS
  -m query    search by MAINTAINER
  -n query    search by PORTNAME
  -a query    search by ONLY_FOR_ARCHS
  -p query    search by PLIST_FILES
  -u query    search by USES
  -x          search only ports marked BROKEN
```

#### Examples:

Find broken USES=go ports:

```sh
$ portgrep -x -u go
databases/cayley:
        BROKEN_i386=    gopkg.in/mgo.v2/bson/json.go:320:7: constant 9007199254740992 overflows int
        --------
        USES=           go:modules
databases/litestream:
        BROKEN_i386=    Build error: constant 9223372036854775807 overflows int
        --------
        USES=           go:modules
databases/mongodb36-tools:
        BROKEN_SSL=     openssl libressl libressl-devel
        --------
        USES=   go localbase
```

Find ports depending on `libcjson`, with 2 lines of context:

```sh
$ portgrep -d libcjson -C2
audio/ocp:

        BUILD_DEPENDS=  xa65:devel/xa65
        LIB_DEPENDS=    libcjson.so:devel/libcjson \
                        libdiscid.so:audio/libdiscid \
                        libid3tag.so:audio/libid3tag \
                        libmad.so:audio/libmad \
                        libogg.so:audio/libogg \
                        libvorbis.so:audio/libvorbis

        USES=           compiler:c11 dos2unix gmake gnome iconv localbase:ldflags \
                        makeinfo ncurses pkgconfig tar:bz2
multimedia/librist:
        LICENSE_FILE=   ${WRKSRC}/COPYING

        LIB_DEPENDS=    libcjson.so:devel/libcjson \
                        libmbedcrypto.so:security/mbedtls

        USES=           localbase:ldflags meson pkgconfig
devel/libcbor:
        LICENSE_FILE=   ${WRKSRC}/LICENSE.md

        LIB_DEPENDS=    libcjson.so:devel/libcjson

        USES=           cmake
net/mosquitto:

        BUILD_DEPENDS=  xsltproc:textproc/libxslt \
                        docbook-xsl>0:textproc/docbook-xsl
        LIB_DEPENDS=    libuuid.so:misc/e2fsprogs-libuuid \
                        libcjson.so:devel/libcjson
        RUN_DEPENDS=    ${LOCALBASE}/share/certs/ca-root-nss.crt:security/ca_root_nss

devel/tinycbor:
        LICENSE_FILE=   ${WRKSRC}/LICENSE

        LIB_DEPENDS=    libcjson.so:devel/libcjson

        USES=           gmake localbase pathfix
```

Search by an arbitrary regex:

```sh
$ portgrep 'REINPLACE_CMD.*\s-i'
www/yarn:

                @${REINPLACE_CMD} -i '' \
                        -e 's|"installationMethod": "tar"|"installationMethod": "pkg"|g' \
                        ${WRKSRC}/package.json

devel/py-os-brick:

                @${GREP} -Rl -Ee '${MY_REGEX}' --null \
                        ${WRKSRC}/etc ${WRKSRC}/os_brick | \
                                ${XARGS} -0 ${REINPLACE_CMD} -i '' -Ee \
                                        "s,${MY_REGEX},${PREFIX}\1,g"

devel/sfml1:

                @${FIND} ${STAGEDIR}${PREFIX}/include/SFML -name "*.hpp" -exec ${REINPLACE_CMD} -i '' -e '/#include/ s|SFML|&1|' {} \;
```

#### Performance

```sh
$ time (find /usr/ports -name Makefile | xargs grep "MAINTAINER=.*ports@" >/dev/null)

real    0m1.045s
user    0m0.237s
sys     0m1.076s
```

```sh
$ time (./portgrep -m ports@ >/dev/null)

real    0m0.395s
user    0m2.238s
sys     0m0.796s
```
