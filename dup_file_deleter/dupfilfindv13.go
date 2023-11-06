package main

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const bufferSize = 20 * 1024 * 1024 // 20MB

func hashFileMD5(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	buf := make([]byte, bufferSize)
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
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func scanDirectory(path string) (map[string][]string, error) {
	fileHashes := make(map[string][]string)
	fileSizes := make(map[int64][]string)
	var mutex sync.Mutex

	tasks := make(chan string, 100) // Buffered channel
	var wg sync.WaitGroup

	log.Println("Starting directory walk...")

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing file %s: %v", filePath, err) // Log the error
			return nil // Return nil to continue walking the directory tree
		}

		if !info.IsDir() {
			log.Printf("Found file: %s", filePath)
			mutex.Lock()
			fileSizes[info.Size()] = append(fileSizes[info.Size()], filePath)
			mutex.Unlock()
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	workerCount := 30
	for i := 0; i < workerCount; i++ {
		wg.Add(1) // Add to the WaitGroup before starting the goroutine
		go func() {
			defer wg.Done() // Mark this worker as done when the function returns
			for file := range tasks {
				hash, err := hashFileMD5(file)
				if err != nil {
					log.Println("Error hashing file:", file, "Error:", err)
					continue
				}
				mutex.Lock()
				if _, exists := fileHashes[hash]; exists {
					log.Println("Duplicate hash found for file:", file, "Hash:", hash)
				}
				fileHashes[hash] = append(fileHashes[hash], file)
				mutex.Unlock()
			}
		}()
	}

	for _, files := range fileSizes {
		if len(files) > 1 {
			for _, file := range files {
				wg.Add(1) // Add to the WaitGroup for each file to be processed
				go func(file string) {
					defer wg.Done() // Mark this file as done when the function returns
					tasks <- file // Feed the task to the channel
				}(file)
			}
		}
	}

	// Close the tasks channel after all tasks have been sent
	go func() {
		wg.Wait()
		close(tasks)
	}()

	return fileHashes, nil
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Enter the target directory to scan: ")
	scanner.Scan()
	targetDir := scanner.Text()

	if _, err := os.Stat(targetDir); os.IsNotExist(err) { // Check if directory exists
		log.Fatalf("The directory %s does not exist", targetDir)
	}

	fmt.Print("Do you want to recursively scan directories? (y/n): ")
	scanner.Scan()
	if scanner.Text() != "y" {
		log.Println("The program currently only supports recursive scanning.")
		return
	}

	fileHashes, err := scanDirectory(targetDir)
	if err != nil {
		log.Println("Error:", err)
		return
	}

	timestamp := time.Now().Format("20060102150405")
	outputFileName := fmt.Sprintf("%s_%s.txt", strings.ReplaceAll(targetDir, string(os.PathSeparator), "_"), timestamp)

	outputFile, err := os.Create(outputFileName)
	if err != nil {
		log.Println("Error creating output file:", err)
		return
	}
	defer outputFile.Close()

	for hash, files := range fileHashes {
		if len(files) > 1 {
			outputFile.WriteString(fmt.Sprintf("Hash: %s\n", hash))
			for _, file := range files {
				outputFile.WriteString(fmt.Sprintf("- %s\n", file))
			}
			outputFile.WriteString("\n")
		}
	}

	log.Printf("Duplicate scan completed. Results written to %s\n", outputFileName)
}
