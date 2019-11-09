package grep

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"
)

var Stop = errors.New("stop")

type Result struct {
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

type GrepFunc func(path string, res Results, err error) error

func Grep(root string, rxs []Regexp, fn GrepFunc, jobs int) error {
	walkPipe, err := walk(root, jobs)
	if err != nil {
		return err
	}
	grepPipe, err := walkPipe.grep(rxs, jobs)
	if err != nil {
		return err
	}

	for x := range grepPipe {
		if err := fn(x.path, x.results, x.err); err != nil {
			if err == Stop {
				break
			}
			return err
		}
	}

	return nil
}

var ignores = map[string]struct{}{
	".svn":      struct{}{},
	".git":      struct{}{},
	"Mk":        struct{}{},
	"Keywords":  struct{}{},
	"Templates": struct{}{},
	"Tools":     struct{}{},
	"distfiles": struct{}{},
	"packages":  struct{}{},
}

type walkResult struct {
	path string
	err  error
}

type walkChan chan walkResult

func walk(root string, jobs int) (walkChan, error) {
	dir, err := ioutil.ReadDir(root)
	if err != nil {
		return nil, err
	}

	out := make(walkChan)

	go func() {
		defer close(out)

		var wg sync.WaitGroup
		sem := make(chan int, jobs)

		for _, fi := range dir {
			if !fi.IsDir() {
				continue
			}

			name := fi.Name()
			if _, ok := ignores[name]; ok {
				continue
			}

			sem <- 1
			wg.Add(1)

			go func(category string) {
				defer func() {
					<-sem
					wg.Done()
				}()

				categoryRoot := filepath.Join(root, category)
				dir, err := ioutil.ReadDir(categoryRoot)
				if err != nil {
					out <- walkResult{err: err}
					return
				}
				for _, fi := range dir {
					if fi.IsDir() {
						out <- walkResult{path: filepath.Join(categoryRoot, fi.Name())}
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

func (walk walkChan) grep(rxs []Regexp, jobs int) (grepChan, error) {
	out := make(grepChan)

	go func() {
		defer close(out)

		var wg sync.WaitGroup
		sem := make(chan int, jobs)

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

				f, err := ioutil.ReadFile(filepath.Join(portRoot, "Makefile"))
				if err != nil {
					out <- grepResult{err: err}
					return
				}
				f = bytes.ReplaceAll(f, []byte{'\\', '\n'}, []byte{0, 0})

				var res Results
				for _, r := range rxs {
					m, err := r.Match(f)
					if err != nil {
						out <- grepResult{err: err}
						return
					}
					if m == nil {
						return
					}
					res = append(res, m)
					// sm := r.re.FindSubmatchIndex(f)
					// if sm == nil {
					// 	return
					// }
					// // fmt.Printf("====>  %q\n", string(f))
					// // fmt.Printf("====>  %s\n", r.re)
					// if len(sm) <= r.rsi {
					// 	out <- grepResult{err: fmt.Errorf("unexpected number of subexpressions: %v", r)}
					// 	return
					// }
					// m := &Match{
					// 	Text: bytes.ReplaceAll(f[sm[0]:sm[1]], []byte{0, 0}, []byte{'\\', '\n'}),
					// 	// Text:           f[sm[0]:sm[1]],
					// 	QuerySubmatch:  []int{sm[2*r.qsi] - sm[0], sm[2*r.qsi+1] - sm[0]},
					// 	ResultSubmatch: []int{sm[2*r.rsi] - sm[0], sm[2*r.rsi+1] - sm[0]},
					// }
					// matches = append(matches, m)
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
