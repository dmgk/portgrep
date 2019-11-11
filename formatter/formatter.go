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

var (
	Cquery  = "\033[0;91m"
	Cresult = "\033[0;92m" // "\033[4m"
)

const creset = "\033[0m"

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
		buf.WriteString(path)
		buf.WriteString(":\n")

		for _, m := range results {
			formatBuf := getBuf()
			defer putBuf(formatBuf)

			if f.flags&Fcolor != 0 {
				formatBuf.Write(m.Text[:m.QuerySubmatch[0]])
				formatBuf.Write([]byte(Cquery))
				formatBuf.Write(m.Text[m.QuerySubmatch[0]:m.QuerySubmatch[1]])
				formatBuf.Write([]byte(creset))
				if m.ResultSubmatch != nil {
					formatBuf.Write(m.Text[m.QuerySubmatch[1]:m.ResultSubmatch[0]])
					formatBuf.Write([]byte(Cresult))
					formatBuf.Write(m.Text[m.ResultSubmatch[0]:m.ResultSubmatch[1]])
					formatBuf.Write([]byte(creset))
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
