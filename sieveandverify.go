// Copyright 2025 Gridbach
package main

import (
	"log"
	"math"
	"math/bits"
)


type MaxElement struct {
	j int
	k int
	r int
}

func SieveAndVerify(jobId uint64) bool {
	log.Printf("SieveAndVerify(%d)", jobId)

	if (root[0] == 0) {
		log.Print("Root is not initialized")
		return false;
	}

	log.Printf("jobId: %d", jobId)
	log.Printf("origin: %d", origin) 
	log.Printf("Sieving from %d to %d", 
			   	origin + uint64(step) * jobId, 
				origin + uint64(step) * (jobId + 1))
	log.Print("Sieving ...")

	// Normalize range for the sake of Goldbach binary verification
	from := origin + uint64(step) * jobId
	to := (from+uint64(step)) - ((from+uint64(step))&0xf) + 15
	from = from - (from&0xf) + 1 - reverseLen<<4

	// Prime array
	var prime = make([]byte, (to-from+2)>>4)

	// Turn all bits on
	for i := 0; i < len(prime); i++ {
		prime[i] = 0xff
	}

	//      b0  b1  b2  b3  b4  b5  b6  b7
	// [i0]  1   3   5   7   9  11  13  15
	// [i1] 17  19  21  23  25  27  29  31
	// [i2] 33  35  37  39  41  45  47  49
	// To access x,
	//  i : (x-1)/16    : x>>4
	//  b : (x%16-1)/2  : (x&15)>>1

	flags := [...]byte{
		0b00000001, 0b00000010, 0b00000100, 0b00001000,
		0b00010000, 0b00100000, 0b01000000, 0b10000000}
	masks := [...]byte{
		0b11111110, 0b11111101, 0b11111011, 0b11110111,
		0b11101111, 0b11011111, 0b10111111, 0b01111111}
	var xmax = uint32(math.Sqrt(float64(to)))
	var yz = uint32(to - from + 1)

	// --- Cache-friendly segmented sieve ---
	//
	// Sieving primes up to xmax (~2e9) are split into two tiers:
	//
	// Tier 1 — small primes (x <= smallPrimeLimit, ~78K primes):
	//   Pre-extracted into a compact slice with one nextMult[] entry each.
	//   prime[] is processed in 32KB chunks so the working set fits in L1 cache.
	//   Each chunk is fully sieved by all small primes before moving on.
	//
	// Tier 2 — large primes (x > smallPrimeLimit):
	//   Each contributes at most a handful of marks to prime[], so the
	//   cache miss cost per prime is low. Handled with the original one-pass
	//   approach — avoids storing nextMult[] for ~98M large primes (~392MB).
	//
	// Crossover: primes larger than the chunk's number span (32KB*16 = 524K)
	// appear at most once per chunk anyway, so chunking them yields no benefit.

	const chunkBytes = 32 * 1024          // 32KB — fits in L1 cache
	const smallPrimeLimit = uint32(1 << 20) // ~1M; ~78K primes below this

	// Step 1: extract small primes into a compact slice
	smallXmax := smallPrimeLimit
	if xmax < smallXmax {
		smallXmax = xmax
	}
	smallPrimes := make([]uint32, 0, 80000)
	for x := uint32(3); x <= smallXmax; x += 2 {
		if root[x>>4]&flags[(x&15)>>1] != 0 {
			smallPrimes = append(smallPrimes, x)
		}
	}

	// Step 2: compute initial next-odd-multiple of each small prime that falls >= from
	nextMult := make([]uint32, len(smallPrimes))
	for i, x := range smallPrimes {
		q, r := bits.Div64(0, from, uint64(x))
		if r != 0 {
			q++
		}
		if q&1 == 0 {
			q++
		}
		nextMult[i] = uint32(uint64(x)*q - from)
	}

	// Step 3: sieve small primes chunk by chunk (working set = one 32KB chunk)
	for chunkStart := 0; chunkStart < len(prime); chunkStart += chunkBytes {
		chunkEnd := chunkStart + chunkBytes
		if chunkEnd > len(prime) {
			chunkEnd = len(prime)
		}
		for i, x := range smallPrimes {
			y := nextMult[i]
			for int(y>>4) < chunkEnd {
				prime[y>>4] &= masks[(y&15)>>1]
				y += x << 1
			}
			nextMult[i] = y
		}
	}

	// Step 4: sieve large primes (original one-pass approach)
	largeXstart := smallPrimeLimit + 1
	if largeXstart&1 == 0 {
		largeXstart++
	}
	for x := largeXstart; x <= xmax; x += 2 {
		if root[x>>4]&flags[(x&15)>>1] != 0 {
			var q, r = bits.Div64(0, from, uint64(x))
			if r != 0 {
				q++
			}
			if q&1 == 0 {
				q++
			}
			var mm = uint64(x) * q
			var ya = uint32(mm - from)
			for y := ya; y < yz; y += x << 1 {
				prime[y>>4] &= masks[(y&15)>>1]
			}
		}
	}

	log.Print("Verifying ...")

	// The loop for verification
	// TODO: add comments for clarity
	var reverseLen = len(reverse[0])
	var ok = 0
	var me MaxElement
	var p int
	var q uint64
	var verified = true
	me.k = 0
	for r := 0; r < 8; r++ {
		for i := reverseLen; i < len(prime)-1; i++ {
			var j = i
			var k = 0
			for ; k < reverseLen; j, k = j-1, k+1 {
				if prime[j]&reverse[r][k] != 0 {
					ok++
					if k > me.k && r == 0 {
						me.j = j
						me.k = k
						me.r = r
						commonbit := prime[me.j] & reverse[me.r][me.k]
						var cl = math.Log2(float64(commonbit))
						p = me.k<<4 + 1 + 2*(7-int(cl))
						q = from + uint64(me.j<<4) + uint64(2*cl)
					}
					break
				}
				if j == 0 {
					verified = false
				}
			}
		}
	}

	log.Printf("Goldbach verification: %t", verified)
	log.Printf("Goldbach partition: (%d, %d)", p, q)

	return true;
}
