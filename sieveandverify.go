// Copyright 2025 Gridbach
package main

import (
	"log"
	"math"
	"math/bits"
	"runtime"
	"sync"
	"time"
)

type MaxElement struct {
	j int
	k int
	r int
}

// nextMultCache carries the next-multiple offset for each prime across jobs.
// Index i corresponds to primeGaps[i] (i.e. the (i+1)-th prime after 3).
var nextMultCache []uint64 // absolute number value of next multiple
var nextMultCacheFrom uint64

// cacheChunk marks one goroutine's slice of nextMultCache.
// Populated during job 1; reused by jobs 2+ for parallel advance and marking.
type cacheChunk struct {
	idx   int    // start index in nextMultCache
	prime uint64 // prime value at idx
}
var cacheChunks []cacheChunk

// clearBits[g] accumulates bits that goroutine g wants to clear in prime[].
// Pre-allocated once; reused each job to avoid GC pressure.
var clearBits [][]byte

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

	// clearMasks[bit] = 1 << bit: the bit to clear in prime[byte].
	// Complement of the AND-masks used in the original code.
	clearMasks := [...]byte{
		0b00000001, 0b00000010, 0b00000100, 0b00001000,
		0b00010000, 0b00100000, 0b01000000, 0b10000000}

	xmax := uint32(math.Sqrt(float64(to)))
	xmaxU := uint64(xmax)

	nw := runtime.NumCPU()
	tSieve := time.Now()

	if nextMultCache == nil || nextMultCacheFrom == 0 {
		// --- Job 1 ---
		// Pass 1 (sequential, ~10 ms): scan primeGaps to count primes ≤ xmax
		// and record per-goroutine chunk boundaries.
		log.Print("[bench] scanning primeGaps for chunk boundaries ...")
		p := uint64(3)
		nPrimes := 0
		for i := 0; i < len(primeGaps); i++ {
			if p > xmaxU { break }
			nPrimes++
			p += 2 * uint64(primeGaps[i])
		}
		chunkSize := nPrimes / nw
		if chunkSize < 1 { chunkSize = 1 }

		cacheChunks = make([]cacheChunk, 0, nw)
		cacheChunks = append(cacheChunks, cacheChunk{0, 3})
		p = uint64(3)
		count := 0
		for i := 0; i < len(primeGaps) && len(cacheChunks) < nw; i++ {
			if p > xmaxU { break }
			count++
			p += 2 * uint64(primeGaps[i])
			if count >= len(cacheChunks)*chunkSize && p <= xmaxU {
				cacheChunks = append(cacheChunks, cacheChunk{count, p})
			}
		}

		// Pass 2 (parallel): build nextMultCache.
		nextMultCache = make([]uint64, nPrimes)
		var wgBuild sync.WaitGroup
		for g, chunk := range cacheChunks {
			wgBuild.Add(1)
			go func(g int, lo int, startP uint64) {
				defer wgBuild.Done()
				hi := nPrimes
				if g+1 < len(cacheChunks) {
					hi = cacheChunks[g+1].idx
				}
				p := startP
				for i := lo; i < hi; i++ {
					q, r := bits.Div64(0, from, p)
					if r != 0 { q++ }
					if q&1 == 0 { q++ }
					nextMultCache[i] = p * q
					if i < len(primeGaps) {
						p += 2 * uint64(primeGaps[i])
					}
				}
			}(g, chunk.idx, chunk.prime)
		}
		wgBuild.Wait()

		// Pre-allocate clearBits buffers.
		clearBits = make([][]byte, len(cacheChunks))
		for g := range clearBits {
			clearBits[g] = make([]byte, len(prime))
		}
	} else {
		// --- Jobs 2+: advance nextMultCache in parallel ---
		log.Print("[bench] advancing nextMultCache ...")
		stepU := uint64(step)
		var wgAdv sync.WaitGroup
		for g, chunk := range cacheChunks {
			wgAdv.Add(1)
			go func(g int, lo int, startP uint64) {
				defer wgAdv.Done()
				hi := len(nextMultCache)
				if g+1 < len(cacheChunks) {
					hi = cacheChunks[g+1].idx
				}
				p := startP
				for i := lo; i < hi; i++ {
					var mm uint64
					if p < stepU {
						q, r := bits.Div64(0, from, p)
						if r != 0 { q++ }
						if q&1 == 0 { q++ }
						mm = p * q
					} else {
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
			}(g, chunk.idx, chunk.prime)
		}
		wgAdv.Wait()
	}
	nextMultCacheFrom = from

	log.Printf("[bench] cache build/update: %d ms", time.Since(tSieve).Milliseconds())

	// --- Parallel marking ---
	// Each goroutine marks its share of primes into clearBits[g] (OR of bit masks),
	// then the merge step ANDs the inverse into prime[].
	tMark := time.Now()

	nBufs := len(cacheChunks)
	// Grow clearBits if prime[] grew (shouldn't normally happen).
	for g := range clearBits {
		if len(clearBits[g]) < len(prime) {
			clearBits[g] = make([]byte, len(prime))
		}
	}

	var wgMark sync.WaitGroup
	for g, chunk := range cacheChunks {
		wgMark.Add(1)
		go func(g int, lo int, startP uint64) {
			defer wgMark.Done()
			hi := len(nextMultCache)
			if g+1 < len(cacheChunks) {
				hi = cacheChunks[g+1].idx
			}
			p := startP
			cb := clearBits[g]
			for i := range cb[:len(prime)] {
				cb[i] = 0
			}
			for i := lo; i < hi; i++ {
				ya := uint32(nextMultCache[i] - from)
				for y := ya; y < yz; y += uint32(p) << 1 {
					cb[y>>4] |= clearMasks[(y&15)>>1]
				}
				if i < len(primeGaps) {
					p += 2 * uint64(primeGaps[i])
				}
			}
		}(g, chunk.idx, chunk.prime)
	}
	wgMark.Wait()

	// Merge: clear the accumulated bits from all goroutines.
	var wgMerge sync.WaitGroup
	mergeChunk := (len(prime) + nBufs - 1) / nBufs
	for g := 0; g < nBufs; g++ {
		wgMerge.Add(1)
		go func(g int) {
			defer wgMerge.Done()
			lo := g * mergeChunk
			hi := lo + mergeChunk
			if hi > len(prime) { hi = len(prime) }
			for i := lo; i < hi; i++ {
				var combined byte
				for h := 0; h < nBufs; h++ {
					combined |= clearBits[h][i]
				}
				prime[i] &= ^combined
			}
		}(g)
	}
	wgMerge.Wait()

	log.Printf("[bench] marking: %d ms", time.Since(tMark).Milliseconds())
	log.Printf("[bench] sieve total: %d ms", time.Since(tSieve).Milliseconds())

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

	log.Printf("Goldbach verification: %t", verified)
	log.Printf("Goldbach partition: (%d, %d)", pp, q)

	return true
}
