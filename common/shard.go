/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"encoding/binary"
)

// LocalShardNumber defines the shard number of coinbase.
// Generally, it must be initialized during program startup.
var LocalShardNumber uint

// IsShardDisabled indicates if the shard is disabled.
// THIS IS FOR UNIT TEST PURPOSE ONLY!!!
var IsShardDisabled = false

// GetShardNumber calculates and returns the shard number for the specified address.
// The valid shard number is [1, ShardNumber], or 0 if IsShardDisabled is true.
func GetShardNumber(address Address) uint {
	if IsShardDisabled {
		return 0
	}

	addrBytes := address.Bytes()
	var sum uint

	// sum [0, 59]
	for _, b := range addrBytes[:60] {
		sum += uint(b)
	}

	// sum [60, 63]
	tail := binary.BigEndian.Uint32(addrBytes[60:])
	sum += uint(tail)

	return (sum % ShardNumber) + 1
}

// CreateContractAddress returns a contract address that in the same shard of the specified address.
func CreateContractAddress(address Address, addrHash, nonceHash []byte) Address {
	if len(addrHash) != HashLength || len(nonceHash) != HashLength {
		panic("invalid hash len")
	}

	targetShardNum := GetShardNumber(address)
	contractAddr := append(addrHash, nonceHash[:28]...) // 32 + 28 bytes
	var sum uint

	// sum [0, 59]
	for _, b := range contractAddr {
		sum += uint(b)
	}

	// sum [60, 63]
	shard := (sum % ShardNumber) + 1
	encoded := make([]byte, 4)

	if shard <= targetShardNum {
		binary.BigEndian.PutUint32(encoded, uint32(targetShardNum-shard))
	} else {
		binary.BigEndian.PutUint32(encoded, uint32(ShardNumber+targetShardNum-shard))
	}

	contractAddr = append(contractAddr, encoded...)

	return BytesToAddress(contractAddr)
}
