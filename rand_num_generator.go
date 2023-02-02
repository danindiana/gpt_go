//Write a program in go which produces durable random numbers from "math/rand" library because it uses a random seed value each time it is called. The numbers should be alphanumeric and at least ten digits in length. The program should print out one hundred random numbers each time it is run.

package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 100; i++ {
		// Generate a 10 digit alphanumeric string
		b := make([]byte, 10)
		for i := range b {
			b[i] = byte(rand.Intn(36))
			if b[i] < 10 {
				b[i] += 48
			} else {
				b[i] += 55
			}
		}
		fmt.Println(string(b))
	}
}
