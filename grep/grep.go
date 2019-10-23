package grep

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"sync"
)

type GrepFunc func(path string, matches [][][]byte, err error) bool

func Grep(root string, rxs []*regexp.Regexp, fn GrepFunc, jobs int) error {
	walkPipe, err := walk(root, jobs)
	if err != nil {
		return err
	}
	grepPipe, err := walkPipe.grep(rxs, jobs)
	if err != nil {
		return err
	}

	for x := range grepPipe {
		if !fn(x.path, x.matches, x.err) {
			break
		}
	}

	return nil
}

var ignores = map[string]struct{}{
	".svn":      struct{}{},
	".git":      struct{}{},
	"distfiles": struct{}{},
	"Mk":        struct{}{},
	"Templates": struct{}{},
	"Tools":     struct{}{},
}

type walkResult struct {
	path string
	err  error
}

type walkChan chan *walkResult

func walk(root string, jobs int) (walkChan, error) {
	rootDir, err := ioutil.ReadDir(root)
	if err != nil {
		return nil, err
	}

	out := make(walkChan)

	go func() {
		defer close(out)

		var wg sync.WaitGroup
		sem := make(chan int, jobs)

		for _, fi := range rootDir {
			if !fi.IsDir() {
				continue
			}

			name := fi.Name()
			if _, ok := ignores[name]; ok {
				continue
			}

			sem <- 1
			wg.Add(1)

			go func(name string) {
				defer func() {
					<-sem
					wg.Done()
				}()

				categoryRoot := filepath.Join(root, name)
				categoryDir, err := ioutil.ReadDir(categoryRoot)
				if err != nil {
					out <- &walkResult{err: err}
					return
				}
				for _, fi := range categoryDir {
					if fi.IsDir() {
						out <- &walkResult{path: filepath.Join(categoryRoot, fi.Name())}
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
	matches [][][]byte
	err     error
}

type grepChan chan *grepResult

func (walk walkChan) grep(rxs []*regexp.Regexp, jobs int) (grepChan, error) {
	out := make(grepChan)

	go func() {
		defer close(out)

		var wg sync.WaitGroup
		sem := make(chan int, jobs)

		for w := range walk {
			if w.err != nil {
				out <- &grepResult{err: w.err}
				continue
			}

			// no regexp provided, everything matches
			if len(rxs) == 0 {
				out <- &grepResult{path: w.path}
				continue
			}

			sem <- 1
			wg.Add(1)

			go func(portPath string) {
				defer func() {
					<-sem
					wg.Done()
				}()

				f, err := ioutil.ReadFile(filepath.Join(portPath, "Makefile"))
				if err != nil {
					out <- &grepResult{err: err}
					return
				}
				f = bytes.ReplaceAll(f, []byte("\\\n"), []byte(""))

				var matches [][][]byte
				for _, r := range rxs {
					mm := r.FindSubmatch(f)
					if mm != nil {
						matches = append(matches, mm)
					}
				}

				if matches != nil {
					out <- &grepResult{
						path:    portPath,
						matches: matches,
					}
				}
			}(w.path)
		}

		wg.Wait()
	}()

	return out, nil
}
