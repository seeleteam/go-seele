/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package util

import (
	"encoding/json"
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

// HashLock is used for hash lock
type HashLock struct {
	Lock sync.Mutex
	Hash common.Hash // used sha256
	Data interface{}
}

// NewHashLock created a new HashLock
func NewHashLock(data interface{}) (*HashLock, error) {
	var hashlock HashLock
	v, err := encode(data)
	if err != nil {
		return nil, err
	}

	hashlock.Hash = v
	hashlock.Data = data
	return &hashlock, nil
}

// encode is used to encode the data to Hash with sha256
func encode(data interface{}) (common.Hash, error) {
	v, err := json.Marshal(data)
	if err != nil {
		return common.EmptyHash, err
	}

	return crypto.MustHash(v), nil
}

// HLock is the hashlock for lock the data
func (hashlock *HashLock) HLock() {
	hashlock.Lock.Lock()
}

// HUnLock is the hashlock for unlock the data
func (hashlock *HashLock) HUnLock() {
	hashlock.Lock.Unlock()
}

// Claim is to claim the data is found
func (hashlock *HashLock) Claim(data interface{}) bool {
	v, err := encode(data)
	if err != nil {
		return false
	}

	if v.Equal(hashlock.Hash) {
		return true
	}

	return false
}
