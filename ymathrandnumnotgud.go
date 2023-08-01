//Write a program in go which attacks the "math/rand" pseudo random library. The output should explain why this library fails to produce durable random numbers.
package main

import (
	"fmt"
	"math/rand"
)

func main() {
	// seed the random number generator
	rand.Seed(1)

	// generate 10 random numbers
	var nums [10]int
	for i := 0; i < 10; i++ {
		nums[i] = rand.Intn(100)
	}

	// print the numbers
	fmt.Println(nums)

	// explain why math/rand fails to produce durable random numbers
	fmt.Println("math/rand fails to produce durable random numbers because it uses a fixed seed value each time it is called.")
	fmt.Println("This means that if you call the same function multiple times, it will produce the same sequence of random numbers.")

	//generate 10 random numbers using a random seed value
	var nums [10]int	
	for i := 0; i < 10; i++ {
		nums[i] = rand.Intn(100)

}	
