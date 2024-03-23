// This version is a Single Input Multiple Output (only one thread feeds paths).

package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type pair struct {
	hash, path string
}

type fileList []string

type results map[string]fileList

func hashFile(path string) (pair, error) {
	file, err := os.Open(path)
	if err != nil {
		return pair{}, err
	}
	defer file.Close()

	hash := md5.New()

	_, err = io.Copy(hash, file)
	if err != nil {
		return pair{}, err
	}

	return pair{string(hash.Sum(nil)), path}, nil
}

func processFile(paths <-chan string, pairs chan<- pair, done chan<- struct{}) {
	defer func() {
		done <- struct{}{}
	}()

	for path := range paths {
		p, _ := hashFile(path)
		pairs <- p
	}
}

func collectHashes(pairs <-chan pair, result chan<- results) {
	hashes := make(results)

	for pair := range pairs {
		hashes[pair.hash] = append(hashes[pair.hash], pair.path)
	}

	result <- hashes
}

func walkDir(dir string, paths chan string) error {
	visit := func(p string, fi os.FileInfo, err error) error {
		if err != nil && err != os.ErrNotExist {
			return err
		}

		if fi.Mode().IsRegular() && fi.Size() > 0 {
			paths <- p
		}

		return nil
	}

	return filepath.Walk(dir, visit)
}

func main() {
	start := time.Now()

	if len(os.Args) < 2 {
		log.Fatal("missing directory name")
	}

	var (
		paths   = make(chan string)
		pairs   = make(chan pair)
		done    = make(chan struct{})
		result  = make(chan results)
		workers = 2 * runtime.GOMAXPROCS(0)
	)

	for i := 0; i < workers; i++ {
		go processFile(paths, pairs, done)
	}

	go collectHashes(pairs, result)

	if err := walkDir(os.Args[1], paths); err != nil {
		log.Fatalf("could not traverse directory tree: %v\n", err)
	}
	close(paths)

	for i := 0; i < workers; i++ {
		<-done
	}

	// signal that all the hashes have been collected (after all the workers are done)
	close(pairs)

	r := <-result
	for _, files := range r {
		if len(files) > 1 {
			fmt.Printf("number of files: %d\nFiles:\n", len(files))

			for _, file := range files {
				fmt.Println(file)
			}
		}
	}

	log.Printf("Total time it took: %s\n", time.Since(start).Round(time.Microsecond))
}
