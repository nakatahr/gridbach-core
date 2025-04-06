// Copyright 2025 Gridbach
package main

import (
	"log"
	"math/bits"
)

func CreateReverse() bool {
	log.Print("CreateReverse()")

	if (root[0] == 0) {
		log.Print("root is not initialized")
		return false;
	}

	// Copy root to source
	// Note that copy() is safe to specify smaller size array as destination
	var source = make([]byte, reverseLen)
	copy(source, root)

	for i := 0; i < len(reverse); i++ {
		reverse[i] = make([]byte, reverseLen)
	}

	for i := 0; i < len(reverse); i++ {
		bumped := byte(0)
		for j := 0; j < len(reverse[i]); j++ {
			x := source[j]
			reverse[i][j] = bits.Reverse8(x<<i | bumped)
			bumped = x >> (8 - i)
		}
	}

	return true
}
