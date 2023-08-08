package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

type fileData struct {
	Path string
	MD5  string
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter the path of the text file containing duplicate file information: ")
	textFilePath, _ := reader.ReadString('\n')
	textFilePath = strings.TrimSpace(textFilePath)

	file, err := os.Open(textFilePath)
	if err != nil {
		fmt.Println("Error opening the text file:", err)
		return
	}
	defer file.Close()

	duplicateFiles := make(map[string][]fileData)

	scanner := bufio.NewScanner(file)
	var currentHash string
	var currentFiles []fileData

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Path: ") {
			path := strings.TrimPrefix(line, "Path: ")
			scanner.Scan() // Read the next line (MD5 Hash)
			hash := strings.TrimPrefix(scanner.Text(), "MD5 Hash: ")

			if currentHash == "" {
				currentHash = hash
			} else if currentHash != hash {
				duplicateFiles[currentHash] = currentFiles
				currentFiles = nil
				currentHash = hash
			}

			currentFiles = append(currentFiles, fileData{Path: path, MD5: hash})
		}
	}

	duplicateFiles[currentHash] = currentFiles

	fmt.Println("Duplicate files found:")
	for _, files := range duplicateFiles {
		if len(files) > 1 {
			for _, file := range files {
				fmt.Printf("Path: %s\nMD5 Hash: %s\n", file.Path, file.MD5)
			}
			fmt.Println("-----")
		}
	}

	fmt.Print("Do you want to delete the duplicate files with longer names or the older ones in case of tie? (y/n): ")
	confirmation, _ := reader.ReadString('\n')
	confirmation = strings.TrimSpace(strings.ToLower(confirmation))
	if confirmation == "y" {
		for _, files := range duplicateFiles {
			if len(files) > 1 {
				longestName := ""
				var oldestModTime time.Time
				var longestPath string

				for _, file := range files {
					fileInfo, err := os.Stat(file.Path)
					if err != nil {
						fmt.Println("Error getting file info:", err)
						continue
					}

					if longestName == "" || len(file.Path) > len(longestName) ||
						(len(file.Path) == len(longestName) && fileInfo.ModTime().Before(oldestModTime)) {
						longestName = file.Path
						oldestModTime = fileInfo.ModTime()
						longestPath = file.Path
					}
				}

				for _, file := range files {
					if file.Path != longestPath {
						err := os.Remove(file.Path)
						if err != nil {
							fmt.Println("Error deleting the file:", err)
						} else {
							fmt.Println("Deleted:", file.Path)
						}
					}
				}
			}
		}
	}

	fmt.Print("Please suggest a file name upon completion: ")
	suggestedFileName, _ := reader.ReadString('\n')
	suggestedFileName = strings.TrimSpace(suggestedFileName)

	fmt.Println("Program completed.")
	fmt.Println("Suggested file name:", suggestedFileName)
}
