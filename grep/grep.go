package grep

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

// Stop is a special value that can be returned by GrepFunc to indicate that
// search needs to be terminated early.
var Stop = errors.New("stop")

// Result describes one search match result.
type Result struct {
	// Text holds the match as a byte slice
	Text []byte
	// QuerySubmatch is a byte index pair identifying the query submatch in Text
	QuerySubmatch []int
	// QuerySubmatch is a byte index pair identifying the result submatch in Text
	ResultSubmatch []int
}

func (r *Result) String() string {
	return fmt.Sprintf("Result {Text: %q, QuerySubmatch:%v, ResultSubmatch:%v}", string(r.Text), r.QuerySubmatch, r.ResultSubmatch)
}

type Results []*Result

// GrepFunc is called for each found match and will be passed the path where
// match was found and a slice of match results.  A Special error value Stop
// can be returned to terminate search early.
type GrepFunc func(path string, res Results, err error) error

// Grep searches port Makefiles, looking for matches described by rxs.  It
// starts looking for Makefiles in root directory, and descends up to two
// levels down (category/port).  If cats slice is not empty, Grep descends only
// to categories listed in cats.  By default, multiple regular expressions in
// rxs are AND-ed together, this can be changed by setting rxsOred to true.
// The search will be run by using up to jobs goroutines, the usual practice is
// to set this to runtime.NumCPU() for the best results.
func Grep(portsRoot string, categories []string, rxs []*Regexp, rxsOred bool, gfn GrepFunc, maxJobs int) error {
	walkCh, err := walk(portsRoot, categories, maxJobs)
	if err != nil {
		return err
	}
	grepCh, err := walkCh.grep(rxs, rxsOred, maxJobs)
	if err != nil {
		return err
	}

	for x := range grepCh {
		if err := gfn(x.path, x.results, x.err); err != nil {
			if err == Stop {
				break
			}
			return err
		}
	}

	return nil
}

var ignores = map[string]struct{}{
	".git":      {},
	".hooks":    {},
	".svn":      {},
	"Keywords":  {},
	"Mk":        {},
	"Templates": {},
	"Tools":     {},
	"distfiles": {},
	"packages":  {},
}

type walkResult struct {
	path string
	err  error
}

type walkChan chan walkResult

func walk(portsRoot string, categories []string, maxJobs int) (walkChan, error) {
	dir, err := ioutil.ReadDir(portsRoot)
	if err != nil {
		return nil, err
	}

	out := make(walkChan)

	go func() {
		defer close(out)

		// prepare category filter set for fast lookup
		catSet := make(map[string]struct{})
		for _, c := range categories {
			catSet[c] = struct{}{}
		}

		var wg sync.WaitGroup
		sem := make(chan int, maxJobs)

		for _, fi := range dir {
			if !fi.IsDir() {
				continue
			}

			name := fi.Name()
			if _, ok := ignores[name]; ok {
				continue
			}

			if len(catSet) != 0 {
				if _, ok := catSet[name]; !ok {
					continue
				}
			}

			sem <- 1
			wg.Add(1)

			go func(cat string) {
				defer func() {
					<-sem
					wg.Done()
				}()

				catRoot := filepath.Join(portsRoot, cat)
				dir, err := ioutil.ReadDir(catRoot)
				if err != nil {
					out <- walkResult{err: err}
					return
				}
				for _, fi := range dir {
					if fi.IsDir() {
						out <- walkResult{path: filepath.Join(catRoot, fi.Name())}
					}
				}
			}(name)
		}

		wg.Wait()
	}()

	return out, nil
}

type grepResult struct {
	path    string
	results Results
	err     error
}

type grepChan chan grepResult

func (walk walkChan) grep(rxs []*Regexp, rxsOr bool, maxJobs int) (grepChan, error) {
	out := make(grepChan)

	go func() {
		defer close(out)

		var wg sync.WaitGroup
		sem := make(chan int, maxJobs)

		for w := range walk {
			if w.err != nil {
				out <- grepResult{err: w.err}
				continue
			}

			// no regexp provided, everything matches
			if len(rxs) == 0 {
				out <- grepResult{path: w.path}
				continue
			}

			sem <- 1
			wg.Add(1)

			go func(portRoot string) {
				defer func() {
					<-sem
					wg.Done()
				}()

				buf, err := readFile(filepath.Join(portRoot, "Makefile"))
				if err != nil {
					if err, ok := err.(*os.PathError); ok && err.Err == syscall.ENOENT {
						// Makefile dosn't exist at path... odd, but okay
						return
					}
					out <- grepResult{err: err}
					return
				}
				defer bufPut(buf)

				b := bytes.ReplaceAll(buf.Bytes(), []byte{'\\', '\n'}, []byte{0, 0})

				var res Results
				for _, r := range rxs {
					m, err := r.Match(b)
					if err != nil {
						out <- grepResult{err: err}
						return
					}
					if !rxsOr && m == nil {
						return // results are ANDed and the current rx doesn't match
					}
					if m != nil {
						m.Text = bytes.ReplaceAll(m.Text, []byte{0, 0}, []byte{'\\', '\n'})
						res = append(res, m)
					}
				}

				if res != nil {
					out <- grepResult{path: portRoot, results: res}
				}
			}(w.path)
		}

		wg.Wait()
	}()

	return out, nil
}

func readFile(filename string) (*bytes.Buffer, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	buf := bufGet()
	buf.Grow(int(fi.Size()) + bytes.MinRead)
	_, err = buf.ReadFrom(f)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func bufGet() *bytes.Buffer {
	return bufPool.Get().(*bytes.Buffer)
}

func bufPut(b *bytes.Buffer) {
	b.Reset()
	bufPool.Put(b)
}
