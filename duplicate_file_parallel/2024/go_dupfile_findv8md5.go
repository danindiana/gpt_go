package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type fileData struct {
	Path string
	Size int64
	Hash string
}

func calculateHash(reader io.Reader) (string, error) {
	hash := md5.New()
	if _, err := io.Copy(hash, bufio.NewReader(reader)); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	return calculateHash(file)
}

func calculateURLHash(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	return calculateHash(resp.Body)
}

func findDuplicates(rootDir string, recursive bool, outputFile *os.File) {
	duplicateFiles := make(map[string][]fileData)
	var mutex sync.Mutex
	var wg sync.WaitGroup

	const maxConcurrentGoroutines = 100
	concurrentGoroutines := make(chan struct{}, maxConcurrentGoroutines)

	var processFile = func(path string, info os.FileInfo) {
		defer wg.Done()
		defer func() { <-concurrentGoroutines }()

		hash, err := calculateFileHash(path)
		if err != nil {
			fmt.Fprintf(outputFile, "Error: %s\n", err)
			return
		}

		size := info.Size()
		mutex.Lock()
		existingFiles, found := duplicateFiles[hash]
		duplicateFiles[hash] = append(existingFiles, fileData{
			Path: path,
			Size: size,
			Hash: hash,
		})
		mutex.Unlock()

		if found {
			for _, f := range duplicateFiles[hash] {
				duplicateEntry := fmt.Sprintf("Path: %s\nSize: %d\nMD5 Hash: %s\n-----\n", f.Path, f.Size, f.Hash)
				fmt.Print(duplicateEntry)
				if outputFile != nil {
					outputFile.WriteString(duplicateEntry)
				}
			}
		}
	}

	var scanDirectory func(string)
	scanDirectory = func(dir string) {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Fprintf(outputFile, "Error: %s\n", err)
				return nil
			}

			if !info.IsDir() {
				wg.Add(1)
				concurrentGoroutines <- struct{}{}
				go processFile(path, info)
			}
			return nil
		})
	}

	if recursive {
		scanDirectory(rootDir)
	} else {
		files, err := os.ReadDir(rootDir)
		if err != nil {
			fmt.Fprintf(outputFile, "Error: %s\n", err)
			return
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			path := filepath.Join(rootDir, file.Name())
			info, err := file.Info()
			if err != nil {
				fmt.Fprintf(outputFile, "Error: %s\n", err)
				continue
			}

			wg.Add(1)
			concurrentGoroutines <- struct{}{}
			go processFile(path, info)
		}
	}

	wg.Wait()
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter the target directory to scan: ")
	targetDir, _ := reader.ReadString('\n')
	targetDir = strings.TrimSpace(targetDir)

	fmt.Print("Do you wish to run the scan recursively? (y/n): ")
	recursiveStr, _ := reader.ReadString('\n')
	recursiveStr = strings.TrimSpace(recursiveStr)
	recursive := strings.ToLower(recursiveStr) == "y"

	fmt.Print("Enter the output file name: ")
	outputFileName, _ := reader.ReadString('\n')
	outputFileName = strings.TrimSpace(outputFileName)

	outputFile, err := os.Create(outputFileName)
	if err != nil {
		fmt.Printf("Error creating the output file: %s\n", err)
		return
	}
	defer outputFile.Close()

	findDuplicates(targetDir, recursive, outputFile)

	fmt.Println("Scan completed. Results have been written to", outputFileName)

	// Example of calculating hash for a URL
	url := "http://example.com/"
	urlHash, err := calculateURLHash(url)
	if err != nil {
		fmt.Printf("Error calculating hash for URL %s: %s\n", url, err)
	} else {
		fmt.Printf("MD5 Hash for URL %s: %s\n", url, urlHash)
	}
}
