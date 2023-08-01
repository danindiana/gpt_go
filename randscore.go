//Write a go program which reads random strings from a text file and determines their degree of randomness.
package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
)

func main() {
	// Open text file
	file, err := os.Open("randchar.txt")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	// Read lines of strings
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// Get string
		str := scanner.Text()

		// Calculate degree of randomness
		randomness := 0
		for i := 0; i < len(str); i++ {
			randomness += int(str[i]) * rand.Intn(i+1)
		}
		
		// Print result
		fmt.Printf("%s: %d\n", strings.ToUpper(str), randomness)
	}
}
