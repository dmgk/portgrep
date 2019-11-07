package formatter

import (
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
	Cresult = "\033[0;92m"
)

const creset = "\033[0m"

type Formatter interface {
	Format(path string, matches grep.Matches) error
}

type textFormatter struct {
	mu sync.Mutex // protects w
	w  io.Writer

	root    string
	flags   int
	needSep bool
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

func (f *textFormatter) Format(path string, matches grep.Matches) error {
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

	if matches != nil {
		buf.WriteString(path)
		buf.WriteString(":\n")

		for _, m := range matches {
			buf.WriteByte('\t')
			if f.flags&Fcolor != 0 {
				buf.Write(m.Text[:m.QuerySubmatch[0]])
				buf.Write([]byte(Cquery))
				buf.Write(m.Text[m.QuerySubmatch[0]:m.QuerySubmatch[1]])
				buf.Write([]byte(creset))
				buf.Write(m.Text[m.QuerySubmatch[1]:m.ResultSubmatch[0]])
				buf.Write([]byte(Cresult))
				buf.Write(m.Text[m.ResultSubmatch[0]:m.ResultSubmatch[1]])
				buf.Write([]byte(creset))
				buf.Write(m.Text[m.ResultSubmatch[1]:])
			} else {
				buf.Write(m.Text)
			}
			buf.WriteByte('\n')
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
