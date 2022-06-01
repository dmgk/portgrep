package formatter

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"sync"

	"github.com/dmgk/portgrep/grep"
)

const (
	Fcolor = 1 << iota
	ForiginsOnly
	ForiginsSingleLine
	FstripRoot

	Fdefaults = FstripRoot
)

var colorMap = map[byte]string{
	'a': "\033[0;30m", // black
	'b': "\033[0;31m", // red
	'c': "\033[0;32m", // green
	'd': "\033[0;33m", // yellow
	'e': "\033[0;34m", // blue
	'f': "\033[0;35m", // magenta
	'g': "\033[0;36m", // cyan
	'h': "\033[0;37m", // white
	'A': "\033[0;90m", // bright black (grey)
	'B': "\033[0;91m", // bright red
	'C': "\033[0;92m", // bright green
	'D': "\033[0;93m", // bright yellow
	'E': "\033[0;94m", // bright blue
	'F': "\033[0;95m", // bright magenta
	'G': "\033[0;96m", // bright cyan
	'H': "\033[0;97m", // bright white
}

const creset = "\033[0m"

const DefaultColors = "BCDA"

const (
	cquery = iota
	cmatch
	cpath
	cseparator

	ncolors
)

var colors [ncolors]string

func SetColors(c string) {
	for i, k := range []byte(c) {
		if v, ok := colorMap[k]; ok && i < len(colors) {
			colors[i] = v
		}
	}
}

type Formatter interface {
	SetIndent(indent string)
	Format(path string, matches grep.Results) error
}

type textFormatter struct {
	mu sync.Mutex // protects w
	w  io.Writer

	root    string
	flags   int
	needSep bool
	indent  string
}

func NewText(w io.Writer, root string, flags int) Formatter {
	f := &textFormatter{
		w:     w,
		root:  root,
		flags: flags,
	}
	if !strings.HasSuffix(root, "/") {
		f.root = f.root + "/"
	}
	return f
}

func (f *textFormatter) SetIndent(indent string) {
	f.indent = indent
}

func (f *textFormatter) Format(path string, results grep.Results) error {
	buf := getBuf()
	defer putBuf(buf)

	if f.flags&FstripRoot != 0 {
		path = strings.TrimPrefix(path, f.root)
	}

	if f.flags&ForiginsSingleLine != 0 {
		if f.needSep {
			buf.WriteByte(' ')
		}
		buf.WriteString(path)
		f.needSep = true
		return f.write(buf)
	}

	if f.flags&ForiginsOnly != 0 {
		buf.WriteString(path)
		buf.WriteByte('\n')
		return f.write(buf)
	}

	if results != nil {
		if f.flags&Fcolor != 0 {
			buf.WriteString(colors[cpath])
			buf.WriteString(path)
			buf.WriteString(creset)
		} else {
			buf.WriteString(path)
		}
		buf.WriteString(":\n")

		for i, m := range results {
			formatBuf := getBuf()
			defer putBuf(formatBuf)

			if i > 0 {
				if f.flags&Fcolor != 0 {
					formatBuf.WriteString(colors[cseparator])
					formatBuf.WriteString("--------\n")
					formatBuf.WriteString(creset)
				} else {
					formatBuf.WriteString("--------\n")
				}
			}

			if f.flags&Fcolor != 0 {
				if m.QuerySubmatch != nil {
					formatBuf.Write(m.Text[:m.QuerySubmatch[0]])
					formatBuf.WriteString(colors[cquery])
					formatBuf.Write(m.Text[m.QuerySubmatch[0]:m.QuerySubmatch[1]])
					formatBuf.WriteString(creset)
				}
				if m.QuerySubmatch != nil && m.ResultSubmatch != nil {
					formatBuf.Write(m.Text[m.QuerySubmatch[1]:m.ResultSubmatch[0]])
				}
				if m.ResultSubmatch != nil {
					formatBuf.WriteString(colors[cmatch])
					formatBuf.Write(m.Text[m.ResultSubmatch[0]:m.ResultSubmatch[1]])
					formatBuf.WriteString(creset)
					formatBuf.Write(m.Text[m.ResultSubmatch[1]:])
				}
			} else {
				formatBuf.Write(m.Text)
			}

			if f.indent != "" {
				sc := bufio.NewScanner(formatBuf)
				for sc.Scan() {
					buf.WriteString(f.indent)
					buf.WriteString(sc.Text())
					buf.WriteByte('\n')
				}
			} else {
				buf.Write(formatBuf.Bytes())
			}
		}

		return f.write(buf)
	}

	return nil
}

func (f *textFormatter) write(buf *bytes.Buffer) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, err := f.w.Write(buf.Bytes())
	return err
}

var bufPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

func getBuf() *bytes.Buffer {
	return bufPool.Get().(*bytes.Buffer)
}

func putBuf(buf *bytes.Buffer) {
	buf.Reset()
	bufPool.Put(buf)
}

func init() {
	SetColors(DefaultColors)
}
