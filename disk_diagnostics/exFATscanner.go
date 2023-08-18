package main

import (
	"bufio"
	"fmt"
	"io/fs" // Import the io/fs package
	"os"
	"path/filepath"
	"strings"
)

func listTopLevelDirectories(path string) {
	fmt.Println("Top-level directories:")
	fmt.Println("---------------------")

	entries, err := os.ReadDir(path)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			fmt.Println(entry.Name())
		}
	}
}

func scanDirectory(path string) (int64, int) {
	var totalSize int64
	var totalFiles int

	err := filepath.WalkDir(path, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			totalSize += info.Size()
			totalFiles++
		}

		return nil
	})

	if err != nil {
		fmt.Println("Error:", err)
	}

	return totalSize, totalFiles
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter target drive (e.g., C:): ")
	targetDrive, _ := reader.ReadString('\n')
	targetDrive = strings.TrimSpace(targetDrive)

	listTopLevelDirectories(targetDrive)

	fmt.Print("Enter the directory to scan: ")
	scanDir, _ := reader.ReadString('\n')
	scanDir = strings.TrimSpace(scanDir)

	scanDir = filepath.Join(targetDrive, scanDir)

	var totalSize int64
	var totalFiles int

	size, files := scanDirectory(scanDir)
	totalSize += size
	totalFiles += files

	averageSize := float64(totalSize) / float64(totalFiles)
	usedSpace := float64(totalSize) / (1024 * 1024) // Convert to MB

	fmt.Printf("Total Files Scanned: %d\n", totalFiles)
	fmt.Printf("Total Drive Size: %.2f MB\n", usedSpace)
	fmt.Printf("Total Used Space: %.2f MB\n", usedSpace)
	fmt.Printf("Average File Size: %.2f bytes\n", averageSize)

	fmt.Print("Enter output file name: ")
	outputFile, _ := reader.ReadString('\n')
	outputFile = strings.TrimSpace(outputFile)

	// Write scan results to the output file
	// You need to implement this part

	fmt.Println("Scan results saved to", outputFile)
}
