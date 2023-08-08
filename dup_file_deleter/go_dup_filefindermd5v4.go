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

func findDuplicates(rootDir string, recursive bool) map[string][]fileData {
	duplicateFiles := make(map[string][]fileData)
	var mutex sync.Mutex

	var scanDirectory func(string)
	scanDirectory = func(dir string) {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Println("Error:", err)
				return nil
			}

			if !info.Mode().IsRegular() {
				return nil
			}

			hash, err := calculateHash(path)
			if err != nil {
				fmt.Println("Error:", err)
				return nil
			}

			size := info.Size()
			mutex.Lock()
			defer mutex.Unlock()
			duplicateFiles[hash] = append(duplicateFiles[hash], fileData{
				Path: path,
				Size: size,
				Hash: hash,
			})

			return nil
		})
	}

	if recursive {
		scanDirectory(rootDir)
	} else {
		files, err := os.ReadDir(rootDir)
		if err != nil {
			fmt.Println("Error:", err)
			return duplicateFiles
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			path := filepath.Join(rootDir, file.Name())
			info, err := file.Info()
			if err != nil {
				fmt.Println("Error:", err)
				continue
			}

			hash, err := calculateHash(path)
			if err != nil {
				fmt.Println("Error:", err)
				continue
			}

			size := info.Size()
			duplicateFiles[hash] = append(duplicateFiles[hash], fileData{
				Path: path,
				Size: size,
				Hash: hash,
			})
		}
	}

	return duplicateFiles
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

	duplicateFiles := findDuplicates(targetDir, recursive)

	fmt.Print("Enter the output file name: ")
	outputFileName, _ := reader.ReadString('\n')
	outputFileName = strings.TrimSpace(outputFileName)

	outputFile, err := os.Create(outputFileName)
	if err != nil {
		fmt.Println("Error creating the output file:", err)
		return
	}
	defer outputFile.Close()

	fmt.Println("Duplicate files found:")
	for _, files := range duplicateFiles {
		if len(files) > 1 {
			for _, file := range files {
				fmt.Println("Path:", file.Path)
				fmt.Println("MD5 Hash:", file.Hash)
				outputFile.WriteString(fmt.Sprintf("Path: %s\nMD5 Hash: %s\n", file.Path, file.Hash))
			}
			fmt.Println("-----")
			outputFile.WriteString("-----\n")
		}
	}

	fmt.Println("Scan completed.")
	fmt.Println("Suggested output file name:", outputFileName)
}
