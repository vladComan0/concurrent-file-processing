package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
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

func walkDir(dir string) (results, error) {
	hashes := make(results)

	visit := func(p string, fi os.FileInfo, err error) error {
		if err != nil && err != os.ErrNotExist {
			return err
		}

		if fi.Mode().IsRegular() && fi.Size() > 0 {
			h, err := hashFile(p)
			if err != nil {
				return err
			}
			hashes[h.hash] = append(hashes[h.hash], h.path)
		}

		return nil
	}

	if err := filepath.Walk(dir, visit); err != nil {
		return results{}, err
	}

	return hashes, nil
}
func main() {
	start := time.Now()

	if len(os.Args) < 2 {
		log.Fatal("missing directory name")
	}

	result, err := walkDir(os.Args[1])
	if err != nil {
		log.Fatalf("could not traverse directory tree: %v\n", err)
	}

	for _, files := range result {
		if len(files) > 1 {
			fmt.Printf("number of files: %d\nFiles:\n", len(files))

			for _, file := range files {
				fmt.Println(file)
			}
		}
	}

	log.Printf("Total time it took: %s\n", time.Since(start).Round(time.Microsecond))
}
