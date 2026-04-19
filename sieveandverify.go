// Copyright 2025 Gridbach
package main

import (
	"log"
	"math"
	"math/bits"
	"time"
)

// Parallelism decision (2026-04-19):
//
// A goroutine-parallel version (N = runtime.NumCPU()) was benchmarked and
// reverted for the following reasons:
//
//  1. Memory-bandwidth-bound, not compute-bound.
//     The cache build/advance reads/writes ~784 MB (98M × 8B) and the marking
//     phase accesses a 6.25 MB prime[] array.  With 7 cores we only achieved
//     1.5–2.4× speedup on individual phases (vs. the theoretical 7×), and
//     only ~1.3× end-to-end (sieve + verify).
//
//  2. WASM target is single-threaded.
//     The webapp runs this code compiled to WASM; runtime.NumCPU() returns 1
//     there, so the parallel paths add complexity with zero benefit for the
//     primary deployment target.
//
//  3. Code complexity outweighs the gain.
//     Separate clearBits buffers, merge passes, cacheChunk bookkeeping, and
//     three WaitGroup phases make the code significantly harder to maintain.
//
// GPU acceleration is the planned path for future parallelism.  When that
// work begins, this file will be the natural starting point.

type MaxElement struct {
	j int
	k int
	r int
}

// nextMultCache carries the next-multiple offset for each prime across jobs.
// Avoids recomputing via 64-bit division on every job after the first.
// Index i corresponds to primeGaps[i] (i.e. the (i+1)-th prime after 3).
var nextMultCache []uint64 // absolute number value of next multiple
var nextMultCacheFrom uint64 // the 'from' value when cache was last updated

func SieveAndVerify(jobId uint64) bool {
	log.Printf("SieveAndVerify(%d)", jobId)

	if len(primeGaps) == 0 {
		log.Print("primeGaps not loaded")
		return false
	}

	log.Printf("Sieving from %d to %d",
		origin+uint64(step)*jobId,
		origin+uint64(step)*(jobId+1))

	from := origin + uint64(step)*jobId
	to := (from+uint64(step)) - ((from+uint64(step))&0xf) + 15
	from = from - (from&0xf) + 1 - reverseLen<<4
	yz := uint32(to - from + 1)

	prime := make([]byte, (to-from+2)>>4)
	for i := range prime {
		prime[i] = 0xff
	}

	masks := [...]byte{
		0b11111110, 0b11111101, 0b11111011, 0b11110111,
		0b11101111, 0b11011111, 0b10111111, 0b01111111}

	xmax := uint32(math.Sqrt(float64(to)))

	tSieve := time.Now()

	// Build or update nextMultCache.
	//
	// nextMultCache[i] = the absolute odd number that is the smallest
	// multiple of prime[i] that is >= from.  Stored as an absolute value
	// (not an offset) so that the advance step for subsequent jobs is a
	// simple nudge (or one division for small primes) rather than a full
	// recomputation from scratch every time.
	//
	// For small primes (p < step), the nudge loop would run step/(2p) times
	// — up to 16M iterations for p=3 — so we recompute via bits.Div64
	// instead.  For large primes (p >= step), the stored value is at most
	// one step behind, so the nudge loop runs 0 or 1 times and is cheap.
	if nextMultCache == nil || nextMultCacheFrom == 0 {
		// First call: compute from scratch via division.
		log.Print("[bench] computing nextMultCache from scratch ...")
		nextMultCache = make([]uint64, len(primeGaps)+1)
		p := uint64(3)
		for i := 0; ; i++ {
			if p > uint64(xmax) {
				nextMultCache = nextMultCache[:i]
				break
			}
			q, r := bits.Div64(0, from, p)
			if r != 0 { q++ }
			if q&1 == 0 { q++ }
			nextMultCache[i] = p * q
			if i < len(primeGaps) {
				p += 2 * uint64(primeGaps[i])
			} else {
				break
			}
		}
	} else {
		// Subsequent jobs: advance nextMultCache to the new 'from'.
		log.Print("[bench] advancing nextMultCache ...")
		p := uint64(3)
		stepU := uint64(step)
		for i := range nextMultCache {
			var mm uint64
			if p < stepU {
				// Small prime: recompute via division (avoids O(step/p) nudge loop).
				q, r := bits.Div64(0, from, p)
				if r != 0 { q++ }
				if q&1 == 0 { q++ }
				mm = p * q
			} else {
				// Large prime: nudge at most once or twice.
				mm = nextMultCache[i]
				for mm < from {
					mm += 2 * p
				}
			}
			nextMultCache[i] = mm
			if i < len(primeGaps) {
				p += 2 * uint64(primeGaps[i])
			}
		}
	}
	nextMultCacheFrom = from

	log.Printf("[bench] cache build/update: %d ms", time.Since(tSieve).Milliseconds())

	// Mark composite numbers in prime[].
	tMark := time.Now()
	p := uint64(3)
	for i, cache := range nextMultCache {
		ya := uint32(cache - from)
		for y := ya; y < yz; y += uint32(p) << 1 {
			prime[y>>4] &= masks[(y&15)>>1]
		}
		if i < len(primeGaps) {
			p += 2 * uint64(primeGaps[i])
		}
	}
	log.Printf("[bench] marking: %d ms", time.Since(tMark).Milliseconds())
	log.Printf("[bench] sieve total: %d ms", time.Since(tSieve).Milliseconds())

	tVerify := time.Now()
	log.Print("Verifying ...")

	var reverseLen = len(reverse[0])
	var ok = 0
	var me MaxElement
	var pp int
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
						pp = me.k<<4 + 1 + 2*(7-int(cl))
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

	log.Printf("[bench] verify: %d ms", time.Since(tVerify).Milliseconds())
	log.Printf("[bench] total (sieve+verify): %d ms", time.Since(tSieve).Milliseconds())
	log.Printf("Goldbach verification: %t", verified)
	log.Printf("Goldbach partition: (%d, %d)", pp, q)

	return true
}
