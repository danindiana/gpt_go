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
	buf := make([]byte, 4096) // 4KB buffer
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return "", err
		}
		if n == 0 {
			break
		}

		hash.Write(buf[:n])
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


func worker(jobs chan job, hashRequests chan<- job, wg *sync.WaitGroup, recursive bool) {
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
					jobs <- job{JobType: scanDir, Path: filepath.Join(j.Path, file.Name())}
				} else if !file.IsDir() {
					hashRequests <- job{JobType: processFile, Path: filepath.Join(j.Path, file.Name()), File: file}
				}
			}
		case processFile:
					info, err := j.File.Info()
					if err != nil {
						fmt.Println("Error:", err)
						continue
					}
					size := info.Size()
					sizeMap[size] = append(sizeMap[size], j)
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

func displayResults(duplicateFiles map[string][]fileData, outputFileName string) {
	fmt.Println("Duplicate files found:")

	outputFile, err := os.Create(outputFileName)
	if err != nil {
		fmt.Println("Error creating the output file:", err)
		return
	}
	defer outputFile.Close()

	writer := bufio.NewWriter(outputFile)
	for _, files := range duplicateFiles {
		if len(files) > 1 {
			for _, file := range files {
				fmt.Println(file.Path)
				writer.WriteString(fmt.Sprintf("MD5: %s\nPath: %s\n\n", file.Hash, file.Path))
			}
			fmt.Println("-----")
		}
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

	jobs := make(chan job, workerCount)
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
		go worker(jobs, hashRequests, &wg, recursive)
	}

	jobs <- job{JobType: scanDir, Path: targetDir}
	wg.Wait()
	close(hashRequests)
	hashWg.Wait()

	displayResults(duplicateFiles, outputFileName)

	fmt.Println("Scan completed!")
}
