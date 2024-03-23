package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"
)

func generateRandom(letterNo int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	b := make([]byte, letterNo)
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	randomString := string(b)

	return randomString
}

func main() {
	dirName := "test"
	if len(os.Args) < 2 {
		log.Fatal("number of files to generate not specified")
	} else if len(os.Args) == 3 {
		dirName = os.Args[2]
	}

	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		if err := os.MkdirAll(dirName, 0755); err != nil {
			log.Fatalf("could not create directory %s due to: %v\n", dirName, err)
		}
	}

	fileNo, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatal("first argument must be a number")
	}

	for i := 0; i < fileNo; i++ {
		fileName := fmt.Sprintf("file_%d", i)
		fullPath := fmt.Sprintf("%s/%s", dirName, fileName)

		f, err := os.Create(fullPath)
		if err != nil {
			log.Printf("could not create file %s due to: %v\n", fileName, err)
			continue
		}

		randomString := generateRandom(4)

		_, err = f.WriteString(randomString)
		if err != nil {
			log.Fatalf("could not write to file %s due to: %v\n", fileName, err)
		}
		f.Close()
	}
}
