package main

import (
	"bufio"
	"context"
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

const bufferSize = 3 * 1024 * 1024 // 1MB

func hashFileMD5(ctx context.Context, path string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		// Continue if the context is not canceled.
	}

	log.Printf("Hashing file: %s\n", path)
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
	result := fmt.Sprintf("%x", hash.Sum(nil))
	log.Printf("Finished hashing file: %s with MD5: %s\n", path, result)
	return result, nil
}

func scanDirectory(ctx context.Context, path string) (map[string][]string, error) {
	fileHashes := make(map[string][]string)
	fileSizes := make(map[int64][]string)
	var mutex sync.Mutex

	tasks := make(chan string, 60) // Buffered channel
	var wg sync.WaitGroup

	log.Println("Starting directory walk...")
	fileCount := 0

	workerCount := 65
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for file := range tasks {
				select {
				case <-ctx.Done():
					log.Printf("Worker %d received cancellation signal\n", id)
					return
				default:
					hash, err := hashFileMD5(ctx, file)
					if err != nil {
						log.Printf("Error hashing file: %s, Error: %v\n", file, err)
						continue
					}
					mutex.Lock()
					fileHashes[hash] = append(fileHashes[hash], file)
					mutex.Unlock()
					log.Printf("Worker %d has processed file: %s\n", id, file)
				}
			}
		}(i)
	}

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing file: %s, Error: %v\n", filePath, err)
			return nil
		}

		if !info.IsDir() {
			log.Printf("Found file: %s", filePath)
			fileCount++
			mutex.Lock()
			fileSizes[info.Size()] = append(fileSizes[info.Size()], filePath)
			mutex.Unlock()
		}
		return nil
	})

	log.Printf("%d files found during walk", fileCount)

	if err != nil {
		close(tasks)
		return nil, err
	}

	// Start enqueuing files to hash
	go func() {
		queuedFileCount := 0
		for _, files := range fileSizes {
			if len(files) > 1 {
				for _, file := range files {
					select {
					case <-ctx.Done():
						log.Println("Cancellation signal received. Stopping enqueue of files")
						close(tasks)
						return
					default:
						tasks <- file
						queuedFileCount++
						log.Printf("File queued for hashing: %s\n", file)
					}
				}
			}
		}
		log.Printf("%d files were queued for hashing", queuedFileCount)
		close(tasks)
	}()

	wg.Wait()
	log.Println("All workers have completed processing")

	return fileHashes, nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Enter the target directory to scan: ")
	scanner.Scan()
	targetDir := scanner.Text()

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		log.Fatalf("The directory %s does not exist", targetDir)
	}

	fmt.Print("Do you want to recursively scan directories? (y/n): ")
	scanner.Scan()
	if scanner.Text() != "y" {
		log.Println("The program currently only supports recursive scanning.")
		return
	}

	start := time.Now()
	fileHashes, err := scanDirectory(ctx, targetDir)
	if err != nil {
		log.Println("Error:", err)
		return
	}
	dur := time.Since(start)
	log.Printf("Scan completed in %s\n", dur)

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
