// Copyright 2025 Gridbach
package main

import (
	"log"
	"math"
)

// global variables
var rootLen uint64 = uint64(3.2 * math.Pow10(9))
var root []byte = make([]byte, int32(math.Ceil(float64(rootLen)/16.0)))
const rootFileName = "root.bin"
var reverseLen = uint64(6256)
var reverse = make([][]byte, 8)
var step = uint32(1 * math.Pow10(8))
var origin = uint64(4 * math.Pow10(18))

func main() {
    if (!LoadRoot()){
        if (!CreateRoot()){
            log.Print("Cannot create or load root file")
        }
    }

    CreateReverse()

    var i uint64
    for i=1; i<=3; i++{
        SieveAndVerify(i)
    }
}
