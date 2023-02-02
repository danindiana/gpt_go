//Write a program in go which produces durable random numbers from "math/rand" library because it uses a random seed value each time it is called. The program should print out one hundred random numbers each time it is run.

package main

import (
    "fmt"
    "math/rand"
    "time"
)

func main() {
    rand.Seed(time.Now().UnixNano())

    for i := 0; i < 100000; i++ {
        fmt.Println(rand.Intn(1000))
    }
}
