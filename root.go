// Copyright 2025 Gridbach
package main

import (
	"log"
	"math"
	"os"
)

func CreateRoot() bool {
	log.Print("CreateRoot()")
	log.Print("Creating. This takes for a minute or two ...")

	for i := 0; i < len(root); i++ {
		root[i] = 0xff
	}
	
	root[0] = 0xfe // obviously 1 is not prime
	
	// Put 0 for the bits after "to"
	for p := 0; p < 8; p++ {
		t := (len(root)-1)*16 + 1 + p*2
		if t > int(rootLen) {
			root[len(root)-1] &= byte(^(1 << p))
		}
	}
	
	//      b0  b1  b2  b3  b4  b5  b6  b7
	// [i0]  1   3   5   7   9  11  13  15
	// [i1] 17  19  21  23  25  27  29  31
	// [i2] 33  35  37  39  41  45  47  49
	// To access x,
	//  i : (x-1)/16    : x>>4
	//  b : (x%16-1)/2  : (x&15)>>1

	x := uint64(3)
	xmax := uint64(math.Sqrt(float64(rootLen)))
	for ; x <= xmax; x += 2 {
		if root[x>>4]&(1<<((x&15)>>1)) != 0 {
			for y := uint64(x * 3); y < rootLen; y += x << 1 {
				mask := byte(^(1 << ((y & 15) >> 1)))
				root[y>>4] &= mask
			}
		}
	}

	// Write a root array to a file
	err := os.WriteFile(rootFileName, root, 0644)
	if err != nil {
		log.Print("Cannot write a file")
		return false;
	}  

	return true;
}

func LoadRoot() bool {
	log.Print("LoadRoot()")

	// Read a root file
	var err error
	root, err = os.ReadFile("root.bin")

	return err == nil
}
