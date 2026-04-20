// Copyright 2025 Gridbach
package main

// sievingprimes.go manages the precomputed list of sieving primes used by
// SieveAndVerify to mark composites in each job's segment.
//
// The sieving primes are all odd primes from 3 up to rootLen, stored as a
// gap-encoded byte slice: each byte holds (gap / 2) between consecutive
// primes. Reconstruct primes by:
//   p[0] = 3
//   p[i] = p[i-1] + 2 * sievingPrimes[i-1]
//
// rootLen is chosen so that sqrt(rootLen²) covers the full verification range.
// At origin ≤ 5×10¹⁸, sqrt(5e18) ≈ 2.236×10⁹, so rootLen = 3.2×10⁹ gives
// comfortable headroom for origins up to ~10¹⁹.
//
// WASM note: sievingprimes.bin can be pre-built, stored in IndexedDB, and
// loaded via LoadSievingPrimes() — BuildSievingPrimes() (which needs ~200 MB
// of working memory) is never called in the browser.

import (
	"log"
	"math"
	"os"
)

// rootLen is the upper bound for the root sieve.
// root[] size = rootLen / 16 = 200,000,000 bytes (200 MB).
const rootLen = 200_000_000 * 16

const sievingPrimesFileName = "sievingprimes.bin"

// sievingPrimes holds the gap-encoded prime list. Populated by
// LoadSievingPrimes or BuildSievingPrimes.
var sievingPrimes []byte

// BuildSievingPrimes runs a full Sieve of Eratosthenes up to rootLen,
// extracts all primes as a gap-encoded list, and saves it to sievingprimes.bin.
//
// The root sieve is allocated locally and freed on return — it is never a
// package-level global.
func BuildSievingPrimes() bool {
	log.Print("BuildSievingPrimes()")
	log.Print("Building root sieve — this may take a minute or two ...")

	// Allocate root sieve locally. Freed when this function returns.
	root := make([]byte, rootLen/16)
	for i := range root {
		root[i] = 0xff
	}
	root[0] = 0xfe // 1 is not prime

	// Clear bits beyond rootLen.
	for b := 0; b < 8; b++ {
		t := (len(root)-1)*16 + 1 + b*2
		if t > rootLen {
			root[len(root)-1] &= byte(^(1 << b))
		}
	}

	// Sieve of Eratosthenes.
	//
	//      b0  b1  b2  b3  b4  b5  b6  b7
	// [i0]  1   3   5   7   9  11  13  15
	// [i1] 17  19  21  23  25  27  29  31
	// To access x: i = x>>4, b = (x&15)>>1
	xmax := uint64(math.Sqrt(float64(rootLen)))
	for x := uint64(3); x <= xmax; x += 2 {
		if root[x>>4]&(1<<((x&15)>>1)) != 0 {
			for y := x * 3; y < uint64(rootLen); y += x << 1 {
				root[y>>4] &= byte(^(1 << ((y & 15) >> 1)))
			}
		}
	}

	// Extract gap-encoded prime list from the root sieve.
	log.Print("Extracting sieving primes ...")
	gaps := make([]byte, 0, 100_000_000)
	prev := uint32(3)
	for x := uint32(5); x < uint32(rootLen); x += 2 {
		if root[x>>4]&(1<<((x&15)>>1)) != 0 {
			gap := (x - prev) / 2
			if gap > 255 {
				log.Printf("gap overflow at x=%d gap=%d", x, gap*2)
				return false
			}
			gaps = append(gaps, byte(gap))
			prev = x
		}
	}

	if err := os.WriteFile(sievingPrimesFileName, gaps, 0644); err != nil {
		log.Printf("Cannot write %s: %v", sievingPrimesFileName, err)
		return false
	}

	log.Printf("BuildSievingPrimes() done: %d primes, file=%s", len(gaps)+1, sievingPrimesFileName)
	sievingPrimes = gaps
	// root goes out of scope here — GC will reclaim the 200 MB.
	return true
}

// LoadSievingPrimes loads the precomputed sieving primes from sievingprimes.bin.
func LoadSievingPrimes() bool {
	data, err := os.ReadFile(sievingPrimesFileName)
	if err != nil {
		return false
	}
	sievingPrimes = data
	log.Printf("LoadSievingPrimes(): loaded %d sieving primes", len(sievingPrimes))
	return true
}
