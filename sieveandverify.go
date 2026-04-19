// Copyright 2025 Gridbach
package main

import (
	"encoding/binary"
	"log"
	"math"
	"math/bits"
	"time"
)

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
			// Yield the mm (minimum multiple) of p satisfying mm >= from.
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
	//
	// Profiling at origin=4e18, step=1e8 showed ~98M primes in nextMultCache:
	//   ~3M   p < yz/2  "multi-mark"  — inner loop runs ≥2 times
	//   ~9.5M p ≥ yz/2  "single-mark" — next multiple is the only one in range
	//   ~85.8M          "dormant"     — ya ≥ yz, next multiple is outside range
	//
	// Skipping dormant primes early and using a direct mark for single-mark
	// primes avoids inner-loop overhead for 97% of primes.
	//
	// Note: loop fusion (advance+mark in one pass) was tried and benchmarked
	// ~33% slower due to the larger loop body breaking CPU IPC/prefetch. The
	// memory-bandwidth savings (~44ms) were outweighed by the IPC loss (~200ms).
	// Separate tight loops remain faster.
	tMark := time.Now()
	halfYz := yz >> 1
	p := uint64(3)
	for i, cache := range nextMultCache {
		ya := uint32(cache - from)
		if ya < yz {
			if uint32(p) >= halfYz {
				// Large prime: next multiple is the only one in range.
				prime[ya>>4] &= masks[(ya&15)>>1]
			} else {
				// Put off bit for every multiple.
				for y := ya; y < yz; y += uint32(p) << 1 {
					prime[y>>4] &= masks[(y&15)>>1]
				}
			}
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

	// Two structural improvements over the original verify loop:
	//
	// 1. Loop order: i outermost, r in the middle, k innermost.
	//    The original had r outermost, so the 6256-byte window
	//    prime[i-reverseLen..i] was loaded from L3 eight separate times.
	//    With i outermost the window is loaded into L1 once and reused
	//    across all 8 r iterations.
	//
	// 2. Word-at-a-time AND: process 8 k values per inner iteration using
	//    uint64 loads + bits.ReverseBytes64.
	//    prime[] is scanned backward (j = i-k decreases) while reverse[r]
	//    is scanned forward (k increases), so byte order is opposite.
	//    Loading 8 bytes of prime and reversing their order makes the bytes
	//    align correctly for a single 64-bit AND with 8 bytes of reverse[r].
	//    reverseLen = 6256 = 782×8, so no tail handling is needed.
	//
	//    First non-zero byte of the AND word gives the smallest matching k
	//    in the chunk: bits.TrailingZeros64(w)/8 is the byte index.

	for i := reverseLen; i < len(prime)-1; i++ {
		for r := 0; r < 8; r++ {
			foundK := -1
			chunks := reverseLen / 8
			for c := 0; c < chunks; c++ {
				k0 := c * 8
				j0 := i - k0
				// 8 prime bytes going backward from j0 (j0, j0-1, ..., j0-7),
				// reversed so byte 0 of primeWord = prime[j0] (pairs with reverse[r][k0]).
				primeWord := bits.ReverseBytes64(binary.LittleEndian.Uint64(prime[j0-7 : j0+1]))
				revWord := binary.LittleEndian.Uint64(reverse[r][k0 : k0+8])
				if w := primeWord & revWord; w != 0 {
					foundK = k0 + bits.TrailingZeros64(w)/8
					break
				}
			}
			if foundK >= 0 {
				ok++
				if r == 0 && foundK > me.k {
					j := i - foundK
					me.j = j
					me.k = foundK
					me.r = r
					commonbit := prime[j] & reverse[r][foundK]
					cl := math.Log2(float64(commonbit))
					pp = me.k<<4 + 1 + 2*(7-int(cl))
					q = from + uint64(me.j<<4) + uint64(2*cl)
				}
			} else {
				verified = false
			}
		}
	}

	log.Printf("[bench] verify: %d ms", time.Since(tVerify).Milliseconds())
	log.Printf("[bench] total (sieve+verify): %d ms", time.Since(tSieve).Milliseconds())
	log.Printf("Goldbach verification: %t", verified)
	log.Printf("Goldbach partition: (%d, %d)", pp, q)

	return true
}
