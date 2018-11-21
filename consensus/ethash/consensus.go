// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package ethash

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"runtime"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/consensus/utils"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto/sha3"
)

//// Various error messages to mark blocks invalid. These should be private to
//// prevent engine specific errors from being referenced in the remainder of the
//// codebase, inherently breaking if the engine is swapped out. Please put common
//// error types into the consensus package.
var (
	errInvalidMixDigest = errors.New("invalid mix digest")
	errInvalidPoW       = errors.New("invalid proof-of-work")
)

// VerifyHeader checks whether a header conforms to the consensus rules of the
// stock Ethereum ethash engine.
func (ethash *Ethash) VerifyHeader(reader consensus.ChainReader, header *types.BlockHeader) error {
	parent := reader.GetHeaderByHash(header.PreviousBlockHash)
	if parent == nil {
		return consensus.ErrBlockInvalidParentHash
	}

	if err := utils.VerifyHeaderCommon(header, parent); err != nil {
		return err
	}

	return ethash.VerifySeal(reader, header)
}

//
//// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
//// concurrently. The method returns a quit channel to abort the operations and
//// a results channel to retrieve the async verifications.
//func (ethash *Ethash) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
//	// If we're running a full engine faking, accept any input as valid
//	if ethash.config.PowMode == ModeFullFake || len(headers) == 0 {
//		abort, results := make(chan struct{}), make(chan error, len(headers))
//		for i := 0; i < len(headers); i++ {
//			results <- nil
//		}
//		return abort, results
//	}
//
//	// Spawn as many workers as allowed threads
//	workers := runtime.GOMAXPROCS(0)
//	if len(headers) < workers {
//		workers = len(headers)
//	}
//
//	// Create a task channel and spawn the verifiers
//	var (
//		inputs = make(chan int)
//		done   = make(chan int, workers)
//		errors = make([]error, len(headers))
//		abort  = make(chan struct{})
//	)
//	for i := 0; i < workers; i++ {
//		go func() {
//			for index := range inputs {
//				errors[index] = ethash.verifyHeaderWorker(chain, headers, seals, index)
//				done <- index
//			}
//		}()
//	}
//
//	errorsOut := make(chan error, len(headers))
//	go func() {
//		defer close(inputs)
//		var (
//			in, out = 0, 0
//			checked = make([]bool, len(headers))
//			inputs  = inputs
//		)
//		for {
//			select {
//			case inputs <- in:
//				if in++; in == len(headers) {
//					// Reached end of headers. Stop sending to workers.
//					inputs = nil
//				}
//			case index := <-done:
//				for checked[index] = true; checked[out]; out++ {
//					errorsOut <- errors[out]
//					if out == len(headers)-1 {
//						return
//					}
//				}
//			case <-abort:
//				return
//			}
//		}
//	}()
//	return abort, errorsOut
//}

//func (ethash *Ethash) verifyHeaderWorker(chain consensus.ChainReader, headers []*types.Header, seals []bool, index int) error {
//	var parent *types.Header
//	if index == 0 {
//		parent = chain.GetHeader(headers[0].ParentHash, headers[0].Height.Uint64()-1)
//	} else if headers[index-1].Hash() == headers[index].ParentHash {
//		parent = headers[index-1]
//	}
//	if parent == nil {
//		return consensus.ErrUnknownAncestor
//	}
//	if chain.GetHeader(headers[index].Hash(), headers[index].Height.Uint64()) != nil {
//		return nil // known block
//	}
//	return ethash.verifyHeader(chain, headers[index], parent, false, seals[index])
//}
//

// VerifySeal implements consensus.Engine, checking whether the given block satisfies
// the PoW difficulty requirements.
func (ethash *Ethash) VerifySeal(reader consensus.ChainReader, header *types.BlockHeader) error {
	return ethash.verifySeal(reader, header, false)
}

// verifySeal checks whether a block satisfies the PoW difficulty requirements,
// either using the usual ethash cache for it, or alternatively using a full DAG
// to make remote mining fast.
func (ethash *Ethash) verifySeal(reader consensus.ChainReader, header *types.BlockHeader, fulldag bool) error {
	// Recompute the digest and PoW values
	number := header.Height

	var witness EthashWitness
	err := common.Deserialize(header.Witness, &witness)
	if err != nil {
		return fmt.Errorf("ethash witness info is disrupted")
	}

	var (
		digest []byte
		result []byte
	)
	// If fast-but-heavy PoW verification was requested, use an ethash dataset
	if fulldag {
		dataset := ethash.dataset(number, true)
		if dataset.generated() {
			digest, result = hashimotoFull(dataset.dataset, sealHash(header).Bytes(), witness.Nonce.Uint64())

			// Datasets are unmapped in a finalizer. Ensure that the dataset stays alive
			// until after the call to hashimotoFull so it's not unmapped while being used.
			runtime.KeepAlive(dataset)
		} else {
			// Dataset not yet generated, don't hang, use a cache instead
			fulldag = false
		}
	}
	// If slow-but-light PoW verification was requested (or DAG not yet ready), use an ethash cache
	if !fulldag {
		cache := ethash.cache(number)

		size := datasetSize(number)
		if ethash.config.PowMode == ModeTest {
			size = 32 * 1024
		}
		digest, result = hashimotoLight(size, cache.cache, sealHash(header).Bytes(), witness.Nonce.Uint64())

		// Caches are unmapped in a finalizer. Ensure that the cache stays alive
		// until after the call to hashimotoLight so it's not unmapped while being used.
		runtime.KeepAlive(cache)
	}
	// Verify the calculated values against the ones provided in the header
	if !bytes.Equal(witness.MixDigest[:], digest) {
		return errInvalidMixDigest
	}
	target := new(big.Int).Div(two256, header.Difficulty)
	if new(big.Int).SetBytes(result).Cmp(target) > 0 {
		return errInvalidPoW
	}

	return nil
}

// Prepare implements consensus.Engine, initializing the difficulty field of a
// header to conform to the ethash protocol. The changes are done inline.
func (ethash *Ethash) Prepare(reader consensus.ChainReader, header *types.BlockHeader) error {
	parent := reader.GetHeaderByHash(header.PreviousBlockHash)
	if parent == nil {
		return consensus.ErrBlockInvalidParentHash
	}

	header.Difficulty = utils.GetDifficult(header.CreateTimestamp.Uint64(), parent)
	return nil
}

// sealHash returns the hash of a block prior to it being sealed.
func sealHash(header *types.BlockHeader) (hash common.Hash) {
	hasher := sha3.NewKeccak256()

	// header info, except witness
	rlp.Encode(hasher, []interface{}{
		header.Height,
		header.Difficulty,
		header.TxHash,
		header.CreateTimestamp,
		header.Creator,
		header.DebtHash,
		header.ExtraData,
		header.PreviousBlockHash,
		header.ReceiptHash,
		header.StateHash,
		header.TxDebtHash,
	})
	hasher.Sum(hash[:0])
	return hash
}
