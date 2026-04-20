// Copyright 2025 Gridbach
package main

import (
	"log"
	"math"
	"math/bits"
)

func CreateReverse() bool {
	log.Print("CreateReverse()")

	// Build a small local sieve covering the first reverseLen bytes
	// (odd numbers up to reverseLen*16 ≈ 100,096).
	source := make([]byte, reverseLen)
	for i := range source {
		source[i] = 0xff
	}
	source[0] = 0xfe // 1 is not prime
	limit := uint64(reverseLen) * 16
	xmax := uint64(math.Sqrt(float64(limit)))
	for x := uint64(3); x <= xmax; x += 2 {
		if source[x>>4]&(1<<((x&15)>>1)) != 0 {
			for y := x * 3; y < limit; y += x << 1 {
				source[y>>4] &= byte(^(1 << ((y & 15) >> 1)))
			}
		}
	}

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
