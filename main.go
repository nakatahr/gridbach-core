// Copyright 2025 Gridbach
package main

import (
	"log"
)

// global variables
const step uint32 = 100_000_000
const origin uint64 = 4_000_000_000_000_000_000
const reverseLen = 6256
var reverse = make([][]byte, 8)

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
