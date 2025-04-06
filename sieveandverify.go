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

	for x := uint32(3); x <= xmax; x += 2 {
		if root[x>>4]&flags[(x&15)>>1] != 0 {
			// Yield the mm (minimum multiple) of x satisfying mm > from
			var q, r = bits.Div64(0, from, uint64(x))
			if r != 0 {
				q++
			}
			if q&1 == 0 {
				q++
			}
			var mm = uint64(x) * q
			var ya = uint32(mm - from)
			// Put off bit for every multiples
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
