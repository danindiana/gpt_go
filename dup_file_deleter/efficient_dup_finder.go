package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
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

type jobType int

const (
	scanDir jobType = iota
	processFile
)

type job struct {
	JobType jobType
	Path    string
	File    os.DirEntry
}

const workerCount = 100

func calculateHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func worker(jobs chan job, results chan<- []fileData, newScanJobs chan<- job, wg *sync.WaitGroup, recursive bool, dirWg *sync.WaitGroup) {
	defer wg.Done()
	duplicateFiles := make(map[string][]fileData)

	for j := range jobs {
		switch j.JobType {
		case scanDir:
			files, err := os.ReadDir(j.Path)
			if err != nil {
				fmt.Println("Error:", err)
				continue
			}

			for _, file := range files {
				if file.IsDir() && recursive {
					dirWg.Add(1)
					newScanJobs <- job{JobType: scanDir, Path: filepath.Join(j.Path, file.Name())}
				} else if !file.IsDir() {
					newScanJobs <- job{JobType: processFile, Path: filepath.Join(j.Path, file.Name()), File: file}
				}
			}
			dirWg.Done()

		case processFile:
			hash, err := calculateHash(j.Path)
			if err != nil {
				fmt.Println("Error:", err)
				continue
			}

			info, err := j.File.Info()
			if err != nil {
				fmt.Println("Error:", err)
				continue
			}

			size := info.Size()
			duplicateFiles[hash] = append(duplicateFiles[hash], fileData{
				Path: j.Path,
				Size: size,
				Hash: hash,
			})
		}
	}

	var duplicates []fileData
	for _, files := range duplicateFiles {
		if len(files) > 1 {
			duplicates = append(duplicates, files...)
		}
	}

	if len(duplicates) > 0 {
		results <- duplicates
	}
}

func handleNewScanJobs(newScanJobs chan job, jobs chan job, dirWg *sync.WaitGroup) {
	defer close(jobs)
	for j := range newScanJobs {
		jobs <- j
	}
	dirWg.Wait()
	close(newScanJobs)
}

func displayResults(results <-chan []fileData, outputFileName string, wg *sync.WaitGroup) {
	defer wg.Done()

	fmt.Println("Duplicate files found:")

	outputFile, err := os.Create(outputFileName)
	if err != nil {
		fmt.Println("Error creating the output file:", err)
		return
	}
	defer outputFile.Close()

	writer := bufio.NewWriter(outputFile)
	for duplicates := range results {
		for _, file := range duplicates {
			fmt.Println(file.Path)
			writer.WriteString(fmt.Sprintf("MD5: %s\nPath: %s\n\n", file.Hash, file.Path))
		}
		fmt.Println("-----")
	}

	writer.Flush()
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter the target disk/directory to scan: ")
	targetDir, _ := reader.ReadString('\n')
	targetDir = strings.TrimSpace(targetDir)

	fmt.Print("Do you wish to run the scan recursively? (y/n): ")
	recursiveStr, _ := reader.ReadString('\n')
	recursiveStr = strings.TrimSpace(recursiveStr)
	recursive := strings.ToLower(recursiveStr) == "y"

	fmt.Print("Enter the output file name: ")
	outputFileName, _ := reader.ReadString('\n')
	outputFileName = strings.TrimSpace(outputFileName)

	results := make(chan []fileData)
	jobs := make(chan job, workerCount)
	newScanJobs := make(chan job, workerCount)

	var wg sync.WaitGroup
	var dirWg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker(jobs, results, newScanJobs, &wg, recursive, &dirWg)
	}

	go handleNewScanJobs(newScanJobs, jobs, &dirWg)

	dirWg.Add(1)
	newScanJobs <- job{JobType: scanDir, Path: targetDir}

	go func() {
		wg.Wait()
		close(results)
	}()

	fmt.Println("Scanning... Press Ctrl+C to stop the scan.")

	wg.Add(1)
	go displayResults(results, outputFileName, &wg)

	wg.Wait()

	fmt.Println("Scan completed.")
	fmt.Println("Results written to:", outputFileName)
}
