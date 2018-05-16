/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"fmt"
	"math/big"
	"math/rand"
	"strconv"
	"time"

	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/miner/pow"
)

var (
	maxInt64 int64 = 9223372036854775807
)

// test difficult adjustment algorithm
// This simulation is also used for deciding initial difficult
func main() {
	var difficult = big.NewInt(90000000)
	var blockTime = uint64(time.Now().Unix())
	var oldDiff *big.Int = big.NewInt(0)
	var parentTime uint64

	for {
		fmt.Printf("%d\n", difficult)
		mine(difficult)
		fmt.Printf("%d\n", difficult)

		parentTime = blockTime
		blockTime = uint64(time.Now().Unix())
		oldDiff.Set(difficult)

		fmt.Printf("%d\n", difficult)
		difficult = GetDifficult(blockTime, parentTime, difficult)
		fmt.Printf("%d\n", difficult)

		fmt.Printf("mined block with block time: %d, parent time: %d, time interval: %d old difficult: %d, new difficult: %d\n",
			blockTime, parentTime, blockTime-parentTime, oldDiff, difficult)
	}

}

func mine(difficult *big.Int) {
	target := pow.GetMiningTarget(difficult)
	r := rand.New(rand.NewSource(time.Now().Unix()))
	seed := r.Int63()
	var nonce int64 = 0
	var hashInt big.Int

	for {
		bytes := []byte(strconv.FormatInt(seed, 10) + strconv.FormatInt(nonce, 10))
		hash := crypto.HashBytes(bytes)
		hashInt.SetBytes(hash.Bytes())

		if hashInt.Cmp(target) <= 0 {
			return
		}

		if nonce == maxInt64 {
			panic("nonce outage")
		}

		nonce++
	}
}

// GetDifficult adjust difficult
func GetDifficult(time uint64, parentTime uint64, parentDifficult *big.Int) *big.Int {
	// algorithm:
	// diff = parentDiff + parentDiff / 8192 * max (1 - (blockTime - parentTime) / 50, -99)
	// target block time is 60 seconds

	//timeInterval := time - parentTime
	interval := (time - parentTime) / 10

	fmt.Printf("interval:%d\n", interval)

	var x *big.Int
	x = big.NewInt(int64(interval))

	big1 := big.NewInt(1)
	x.Sub(big1, x)
	fmt.Printf("x:%d\n", x)

	big99 := big.NewInt(-99)
	if x.Cmp(big99) < 0 {
		x = big99
	}

	fmt.Printf("x:%d\n", x)

	var y = new(big.Int).Set(parentDifficult)
	big8192 := big.NewInt(2048)
	y.Div(parentDifficult, big8192)

	fmt.Printf("y:%d\n", y)

	var result = big.NewInt(0)
	result.Mul(x, y)
	result.Add(parentDifficult, result)

	return result
}
