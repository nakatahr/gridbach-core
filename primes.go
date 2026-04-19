// Copyright 2025 Gridbach
package main

// BuildPrimes extracts all primes from root[] up to rootLen and stores them
// as a compact gap-encoded list in primegaps.bin.
//
// Encoding: each byte stores (gap / 2) between consecutive primes, starting
// from the first prime after 2 (i.e. 3). Gap/2 always fits in uint8 because
// the maximum prime gap below 3.2e9 is well under 510.
//
// The first prime (3) is implicit. Reconstruct primes by:
//   p[0] = 3
//   p[i] = p[i-1] + 2 * primegaps[i-1]
//
// File size: ~98M bytes for primes up to 2e9 (vs 200MB for root[]).
// Used by SieveAndVerify to avoid scanning root[] per job.

import (
	"log"
	"os"
)

const primegapsFileName = "primegaps.bin"

// primeGaps holds the loaded gap list. Populated by LoadPrimeGaps or BuildPrimes.
var primeGaps []byte

func BuildPrimes() bool {
	log.Print("BuildPrimes()")
	log.Print("Extracting prime gaps from root — this may take a moment ...")

	if root[0] == 0 {
		log.Print("root is not initialized")
		return false
	}

	flags := [...]byte{
		0b00000001, 0b00000010, 0b00000100, 0b00001000,
		0b00010000, 0b00100000, 0b01000000, 0b10000000}

	gaps := make([]byte, 0, 100_000_000)
	prev := uint32(3)
	for x := uint32(5); x < uint32(rootLen); x += 2 {
		if root[x>>4]&flags[(x&15)>>1] != 0 {
			gap := (x - prev) / 2
			if gap > 255 {
				log.Printf("gap overflow at x=%d gap=%d", x, gap*2)
				return false
			}
			gaps = append(gaps, byte(gap))
			prev = x
		}
	}

	err := os.WriteFile(primegapsFileName, gaps, 0644)
	if err != nil {
		log.Printf("Cannot write %s: %v", primegapsFileName, err)
		return false
	}

	log.Printf("BuildPrimes() done: %d primes, file=%s", len(gaps)+1, primegapsFileName)
	primeGaps = gaps
	return true
}

func LoadPrimeGaps() bool {
	data, err := os.ReadFile(primegapsFileName)
	if err != nil {
		return false
	}
	primeGaps = data
	log.Printf("LoadPrimeGaps(): loaded %d prime gaps", len(primeGaps))
	return true
}
