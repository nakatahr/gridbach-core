// Copyright 2025 Gridbach
package main

import (
	"log"
)

// global variables
var reverseLen = uint64(6256)
var reverse = make([][]byte, 8)
var step = uint32(1e8)
var origin = uint64(4e18)

func main() {
	if !LoadSievingPrimes() {
		if !BuildSievingPrimes() {
			log.Print("Cannot create or load sieving primes")
			return
		}
	}

	CreateReverse()

	var i uint64
	for i = 1; i <= 3; i++ {
		SieveAndVerify(i)
	}
}
