//Write a program in go language which outputs a merkle tree  from a random seed. The program should ask the user to input a string of random characters in order to generate said tree. The program should write the output to a text file called merkle.txt when it is run.

package main

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	// Get user input
	fmt.Print("Enter a string of random characters: ")
	var seed string
	fmt.Scanln(&seed)

	// Generate a Merkle Tree from the seed
	root := generateMerkleTree(seed)

	// Write output to a text file
	file, err := os.Create("merkle.txt")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	_, err = file.WriteString(root.String())
	if err != nil {
		panic(err)
	}

	fmt.Println("Merkle Tree written to merkle.txt")
}

// MerkleTree is a structure for representing a Merkle Tree
type MerkleTree struct {
	Left  *MerkleTree
	Right *MerkleTree
	Data  []byte
}

// String returns the string representation of a Merkle Tree
func (m *MerkleTree) String() string {
	if m.Left == nil && m.Right == nil {
		return fmt.Sprintf("%x", m.Data)
	}

	return fmt.Sprintf("%x (%s %s)", m.Data, m.Left.String(), m.Right.String())
}

// generateMerkleTree takes a seed string and returns a Merkle Tree
func generateMerkleTree(seed string) *MerkleTree {
	// Convert seed string to a byte array
	data := []byte(seed)

	// Calculate the hash of the seed
	hash := sha256.Sum256(data)

	// Create a new Merkle Tree with the hash as the root node
	root := &MerkleTree{Data: hash[:]}

	// If the seed is only one character, return the tree
	if len(data) == 1 {
		return root
	}

	// Split the seed string into left and right halves
	mid := len(data) / 2
	left := data[:mid]
	right := data[mid:]

	// Create left and right subtrees
	root.Left = generateMerkleTree(string(left))
	root.Right = generateMerkleTree(string(right))

	// Calculate the hash of the root node
	b := append(root.Left.Data, root.Right.Data...)
	root.Data = sha256.Sum256(b)[:]

	return root
}
