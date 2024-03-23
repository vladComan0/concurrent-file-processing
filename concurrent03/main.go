// This version will create a goroutine for each file on disk.
// However, GOMAXPROCS only takes into account CPU-bound work, not IO-bound work.
// When we have lots of delays due to disk (or network) read/writes,
// additional threads will be created and there is a limit (typically 1000 threads) after which the OS kills the process.
// So we will limit the number of syscalls goroutines using a counting semaphore (with a buffered channel).
// A buffered channel will limit the work in progress (number of workers active at a given time).
// This allows us to control how many goroutines are actively hitting the disk.

package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
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

func processFile(path string, pairs chan<- pair, limit chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	limit <- struct{}{} // attempts to consume one goroutine spot from the counting semaphore
	defer func() {
		<-limit // signal done at the end
	}()

	p, _ := hashFile(path)
	pairs <- p
}

func collectHashes(pairs <-chan pair, result chan<- results) {
	hashes := make(results)

	for pair := range pairs {
		hashes[pair.hash] = append(hashes[pair.hash], pair.path)
	}

	result <- hashes
}

func walkDir(dir string, pairs chan pair, limit chan struct{}, wg *sync.WaitGroup) error {
	defer wg.Done()

	visit := func(p string, fi os.FileInfo, err error) error {
		if err != nil && err != os.ErrNotExist {
			return err
		}

		if fi.IsDir() && p != dir {
			wg.Add(1)
			go walkDir(p, pairs, limit, wg)
			return filepath.SkipDir
		}

		if fi.Mode().IsRegular() && fi.Size() > 0 {
			wg.Add(1)
			go processFile(p, pairs, limit, wg)
		}

		return nil
	}

	limit <- struct{}{} // attempt to start the goroutine if it fits in the buffer, otherwise blocks until it's room for it
	defer func() {
		<-limit // after the work is done, free up a space in the buffered channel
	}()

	return filepath.Walk(dir, visit) // visit closure is called only here, even if it's defined above
}

func main() {
	start := time.Now()

	if len(os.Args) < 2 {
		log.Fatal("missing directory name")
	}

	var (
		workers = 3 * runtime.GOMAXPROCS(0)
		pairs   = make(chan pair)
		result  = make(chan results)
		limit   = make(chan struct{}, workers) // Counting semaphore buffered with the number of workers that should be active on syscalls at a given time
		wg      = new(sync.WaitGroup)
	)

	go collectHashes(pairs, result)

	wg.Add(1)
	if err := walkDir(os.Args[1], pairs, limit, wg); err != nil {
		log.Fatalf("could not traverse directory tree: %v\n", err)
	}
	wg.Wait()

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
