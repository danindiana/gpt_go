package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var linkFilePath string
	fmt.Print("Enter the path to the text document containing the links: ")
	fmt.Scanln(&linkFilePath)

	targetDirectory := "downloads" // Directory where the downloaded files will be saved

	// Create the target directory if it doesn't exist
	if err := os.MkdirAll(targetDirectory, 0755); err != nil {
		fmt.Printf("Error creating target directory: %v\n", err)
		return
	}

	// Open the link file for reading
	file, err := os.Open(linkFilePath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	// Read each line of the link file and download the corresponding document
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		link := strings.TrimSpace(scanner.Text())
		if link == "" {
			continue // Skip empty lines
		}

		fileName := getFileName(link)
		filePath := filepath.Join(targetDirectory, fileName+".pdf") // Append '.pdf' to the file name

		// Download the file
		err := downloadFile(link, filePath)
		if err != nil {
			fmt.Printf("Error downloading file from %s: %v\n", link, err)
			continue
		}
		fmt.Printf("Downloaded %s\n", filePath)
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading file: %v\n", err)
	}
}

func downloadFile(url string, filePath string) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// Create the file to write the downloaded content
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Copy the content from the HTTP response body to the file
	_, err = io.Copy(file, response.Body)
	return err
}

func getFileName(url string) string {
	// Extract the file name from the URL
	tokens := strings.Split(url, "/")
	return tokens[len(tokens)-1]
}
