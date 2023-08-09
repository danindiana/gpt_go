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
const hasherCount = 20

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

func hasher(hashRequests chan job, wg *sync.WaitGroup, duplicateFiles map[string][]fileData, mu *sync.Mutex) {
	defer wg.Done()

	for j := range hashRequests {
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
		key := fmt.Sprintf("%d-%s", size, hash)

		mu.Lock()
		duplicateFiles[key] = append(duplicateFiles[key], fileData{
			Path: j.Path,
			Size: size,
			Hash: hash,
		})
		mu.Unlock()
	}
}

func worker(jobs chan job, results chan<- []fileData, newScanJobs chan<- job, hashRequests chan<- job, wg *sync.WaitGroup, recursive bool) {
	defer wg.Done()
	sizeMap := make(map[int64][]job)

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
					newScanJobs <- job{JobType: scanDir, Path: filepath.Join(j.Path, file.Name())}
				} else if !file.IsDir() {
					jobData := job{JobType: processFile, Path: filepath.Join(j.Path, file.Name()), File: file}
					fileInfo, err := file.Info()
					if err != nil {
					    fmt.Println("Error getting file info:", err)
					    continue
					}
					sizeMap[fileInfo.Size()] = append(sizeMap[fileInfo.Size()], jobData)
				}
			}
		case processFile:
			// Handled in hasher.
		}
	}

	for _, jobsWithSameSize := range sizeMap {
		if len(jobsWithSameSize) > 1 {
			for _, j := range jobsWithSameSize {
				hashRequests <- j
			}
		}
	}
}

func handleNewScanJobs(newScanJobs chan job, jobs chan job) {
	for j := range newScanJobs {
		jobs <- j
	}
	close(jobs)
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
	hashRequests := make(chan job, hasherCount)

	var wg sync.WaitGroup
	var hashWg sync.WaitGroup

	duplicateFiles := make(map[string][]fileData)
	var mu sync.Mutex

	for i := 0; i < hasherCount; i++ {
		hashWg.Add(1)
		go hasher(hashRequests, &hashWg, duplicateFiles, &mu)
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker(jobs, results, newScanJobs, hashRequests, &wg, recursive)
	}

	go handleNewScanJobs(newScanJobs, jobs)

	newScanJobs <- job{JobType: scanDir, Path: targetDir}
	close(newScanJobs)

	go func() {
		wg.Wait()
		close(hashRequests)
		hashWg.Wait()
		for _, files := range duplicateFiles {
			if len(files) > 1 {
				results <- files
			}
		}
		close(results)
	}()


	var displayWg sync.WaitGroup
	displayWg.Add(1)
	go displayResults(results, outputFileName, &displayWg)
	displayWg.Wait()

	fmt.Println("Scan completed!")
}
