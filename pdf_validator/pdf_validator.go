package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	
)

func validatePDF(filePath string) bool {
	err := api.ValidateFile(filePath, nil)
	if err != nil {
		return false
	}
	return true
}

func deleteFiles(filePaths []string) int {
	deletedCount := 0
	for _, filePath := range filePaths {
		err := os.Remove(filePath)
		if err != nil {
			fmt.Printf("Error deleting %s: %v\n", filePath, err)
		} else {
			fmt.Printf("Deleted: %s\n", filePath)
			deletedCount++
		}
	}
	return deletedCount
}

func scanAndValidatePDFFiles(directoryPath string, deleteInvalid, recursive bool) ([]string, []string, int) {
	validFiles := []string{}
	invalidFiles := []string{}

	err := filepath.Walk(directoryPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && !recursive && path != directoryPath {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".pdf") {
			fmt.Printf("Validating: %s\n", path)
			if validatePDF(path) {
				validFiles = append(validFiles, path)
				fmt.Println("  Valid")
			} else {
				invalidFiles = append(invalidFiles, path)
				fmt.Println("  Invalid")
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error scanning directory: %v\n", err)
	}

	deletedCount := 0
	if deleteInvalid {
		deletedCount = deleteFiles(invalidFiles)
	}

	return validFiles, invalidFiles, deletedCount
}

func main() {
	var targetDirectory string
	var recursiveOption, deleteOption string

	fmt.Print("Enter the target directory path: ")
	fmt.Scanln(&targetDirectory)

	fmt.Print("Do you want to scan recursively? (Y/N): ")
	fmt.Scanln(&recursiveOption)
	recursiveScan := strings.TrimSpace(strings.ToLower(recursiveOption)) == "y"

	fmt.Print("Do you want to delete invalid/corrupted files? (Y/N): ")
	fmt.Scanln(&deleteOption)
	deleteInvalid := strings.TrimSpace(strings.ToLower(deleteOption)) == "y"

	fmt.Println("\nScanning and validating PDF files...")
	validFiles, invalidFiles, deletedCount := scanAndValidatePDFFiles(targetDirectory, deleteInvalid, recursiveScan)

	fmt.Println("\nValid PDF files:")
	for _, filePath := range validFiles {
		fmt.Println(filePath)
	}

	fmt.Println("\nInvalid PDF files:")
	for _, filePath := range invalidFiles {
		fmt.Println(filePath)
	}

	suggestedFileName := "validated_files.txt"
	file, err := os.Create(suggestedFileName)
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()

	file.WriteString("Valid PDF files:\n")
	for _, path := range validFiles {
		file.WriteString(path + "\n")
	}
	file.WriteString("\nInvalid PDF files:\n")
	for _, path := range invalidFiles {
		file.WriteString(path + "\n")
	}

	fmt.Printf("\nSuggested file '%s' with the list of valid and invalid PDF files.\n", suggestedFileName)

	if deleteInvalid {
		fmt.Printf("\nNumber of files deleted: %d\n", deletedCount)
	}
}
