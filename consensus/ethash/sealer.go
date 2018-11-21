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
	crand "crypto/rand"
	"encoding/json"
	"errors"
	"math"
	"math/big"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/core/types"
)

const (
	// staleThreshold is the maximum depth of the acceptable stale but valid ethash solution.
	staleThreshold = 7
)

var (
	errNoMiningWork      = errors.New("no mining work available yet")
	errInvalidSealResult = errors.New("invalid or stale proof-of-work solution")
)

type EthashWitness struct {
	Nonce     BlockNonce
	MixDigest common.Hash
}

// Seal implements consensus.Engine, attempting to find a nonce that satisfies
// the block's difficulty requirements.
func (ethash *Ethash) Seal(reader consensus.ChainReader, block *types.Block, stop <-chan struct{}, results chan<- *types.Block) error {
	// Create a runner and the multiple search threads it directs
	abort := make(chan struct{})

	ethash.lock.Lock()
	threads := ethash.threads
	if ethash.rand == nil {
		seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
		if err != nil {
			ethash.lock.Unlock()
			return err
		}
		ethash.rand = rand.New(rand.NewSource(seed.Int64()))
	}
	ethash.lock.Unlock()
	if threads == 0 {
		threads = runtime.NumCPU()
	}
	if threads < 0 {
		threads = 0 // Allows disabling local mining without extra logic around local/remote
	}
	// Push new work to remote sealer
	if ethash.workCh != nil {
		ethash.workCh <- &sealTask{block: block, results: results}
	}
	var (
		pend   sync.WaitGroup
		locals = make(chan *types.Block)
	)
	for i := 0; i < threads; i++ {
		pend.Add(1)
		go func(id int, nonce uint64) {
			defer pend.Done()
			ethash.mine(block, id, nonce, abort, locals)
		}(i, uint64(ethash.rand.Int63()))
	}
	// Wait until sealing is terminated or a nonce is found
	go func() {
		var result *types.Block
		select {
		case <-stop:
			// Outside abort, stop all miner threads
			close(abort)
		case result = <-locals:
			// One of the threads found a block, abort all others
			select {
			case results <- result:
			default:
				ethash.log.Warn("Sealing result is not read by miner. mode: local, seelhash:%s", sealHash(block.Header))
			}
			close(abort)
		case <-ethash.update:
			// Thread count was changed on user request, restart
			close(abort)
			if err := ethash.Seal(reader, block, stop, results); err != nil {
				ethash.log.Error("Failed to restart sealing after update, err %s", err)
			}
		}
		// Wait for all miners to terminate and return the block
		pend.Wait()
	}()
	return nil
}

// mine is the actual proof-of-work miner that searches for a nonce starting from
// seed that results in correct final block difficulty.
func (ethash *Ethash) mine(block *types.Block, id int, seed uint64, abort chan struct{}, found chan *types.Block) {
	// Extract some data from the header
	var (
		header  = block.Header
		hash    = sealHash(header).Bytes()
		target  = new(big.Int).Div(two256, header.Difficulty)
		number  = header.Height
		dataset = ethash.dataset(number, false)
	)
	// Start generating random nonces until we abort or find a good one
	var (
		attempts = int64(0)
		nonce    = seed
	)

	ethash.log.Debug("Started ethash search for new nonces. seed %d", seed)
search:
	for {
		select {
		case <-abort:
			// Mining terminated, update stats and abort
			ethash.log.Debug("Ethash nonce search aborted. attempts %d", nonce-seed)
			ethash.hashrate.Mark(attempts)
			break search

		default:
			// We don't have to update hash rate on every nonce, so update after after 2^X nonces
			attempts++
			if (attempts % (1 << 15)) == 0 {
				ethash.hashrate.Mark(attempts)
				attempts = 0
			}
			// Compute the PoW value of this nonce
			digest, result := hashimotoFull(dataset.dataset, hash, nonce)
			if new(big.Int).SetBytes(result).Cmp(target) <= 0 {
				// Correct nonce found, create a new header with it
				header = header.Clone()
				SetWitness(header, nonce, digest)

				// Seal and return a block (if still needed)
				select {
				case found <- block.WithSeal(header):
					ethash.log.Debug("Ethash nonce found and reported. attempts %d, nonce %d", nonce-seed, nonce)
				case <-abort:
					ethash.log.Debug("Ethash nonce found but discarded. attempts %d, nonce %d", nonce-seed, nonce)
				}
				break search
			}
			nonce++
		}
	}
	// Datasets are unmapped in a finalizer. Ensure that the dataset stays live
	// during sealing so it's not unmapped while being read.
	runtime.KeepAlive(dataset)
}

func SetWitness(header *types.BlockHeader, nonce uint64, digest []byte) {
	SetWitnessStraight(header, EncodeNonce(nonce), common.BytesToHash(digest))
}

func SetWitnessStraight(header *types.BlockHeader, nonce BlockNonce, mixDigest common.Hash) {
	witness := EthashWitness{
		Nonce:     nonce,
		MixDigest: mixDigest,
	}

	header.Witness = common.SerializePanic(witness)
}

// remote is a standalone goroutine to handle remote mining related stuff.
func (ethash *Ethash) remote(notify []string, noverify bool) {
	var (
		works = make(map[common.Hash]*types.Block)
		rates = make(map[common.Hash]hashrate)

		results      chan<- *types.Block
		currentBlock *types.Block
		currentWork  [3]string

		notifyTransport = &http.Transport{}
		notifyClient    = &http.Client{
			Transport: notifyTransport,
			Timeout:   time.Second,
		}
		notifyReqs = make([]*http.Request, len(notify))
	)
	// notifyWork notifies all the specified mining endpoints of the availability of
	// new work to be processed.
	notifyWork := func() {
		work := currentWork
		blob, _ := json.Marshal(work)

		for i, url := range notify {
			// Terminate any previously pending request and create the new work
			if notifyReqs[i] != nil {
				notifyTransport.CancelRequest(notifyReqs[i])
			}
			notifyReqs[i], _ = http.NewRequest("POST", url, bytes.NewReader(blob))
			notifyReqs[i].Header.Set("Content-Type", "application/json")

			// Push the new work concurrently to all the remote nodes
			go func(req *http.Request, url string) {
				res, err := notifyClient.Do(req)
				if err != nil {
					ethash.log.Warn("Failed to notify remote miner. err %s", err)
				} else {
					ethash.log.Debug("Notified remote miner. miner %s. target %s", url, work[2])
					res.Body.Close()
				}
			}(notifyReqs[i], url)
		}
	}
	// makeWork creates a work package for external miner.
	//
	// The work package consists of 3 strings:
	//   result[0], 32 bytes hex encoded current block header pow-hash
	//   result[1], 32 bytes hex encoded seed hash used for DAG
	//   result[2], 32 bytes hex encoded boundary condition ("target"), 2^256/difficulty
	makeWork := func(block *types.Block) {
		hash := sealHash(block.Header)

		currentWork[0] = hash.Hex()
		currentWork[1] = common.BytesToHash(SeedHash(block.Header.Height)).Hex()
		currentWork[2] = common.BytesToHash(new(big.Int).Div(two256, block.Header.Difficulty).Bytes()).Hex()

		// Trace the seal work fetched by remote sealer.
		currentBlock = block
		works[hash] = block
	}
	// submitWork verifies the submitted pow solution, returning
	// whether the solution was accepted or not (not can be both a bad pow as well as
	// any other error, like no pending work or stale mining result).
	submitWork := func(nonce BlockNonce, mixDigest common.Hash, sealhash common.Hash) bool {
		if currentBlock == nil {
			ethash.log.Error("Pending work without block. sealhash %s", sealhash)
			return false
		}
		// Make sure the work submitted is present
		block := works[sealhash]
		if block == nil {
			ethash.log.Warn("Work submitted but none pending. sealhash %s, curheight %d", sealhash, currentBlock.Header.Height)
			return false
		}
		// Verify the correctness of submitted result.
		header := block.Header
		SetWitnessStraight(header, nonce, mixDigest)

		start := time.Now()
		if !noverify {
			if err := ethash.verifySeal(nil, header, true); err != nil {
				ethash.log.Warn("Invalid proof-of-work submitted. sealhash %s. elapsed %s. err %s", sealhash, time.Since(start), err)
				return false
			}
		}
		// Make sure the result channel is assigned.
		if results == nil {
			ethash.log.Warn("Ethash result channel is empty, submitted mining result is rejected")
			return false
		}
		ethash.log.Debug("Verified correct proof-of-work. sealhash %s. elapsed %s", sealhash, time.Since(start))

		// Solutions seems to be valid, return to the miner and notify acceptance.
		solution := block.WithSeal(header)

		// The submitted solution is within the scope of acceptance.
		if solution.Header.Height+staleThreshold > currentBlock.Header.Height {
			select {
			case results <- solution:
				ethash.log.Debug("Work submitted is acceptable. height %d. sealhash %s. hash %s", solution.Header.Height, sealhash, solution.HeaderHash)
				return true
			default:
				ethash.log.Warn("Sealing result is not read by miner. mode remote. sealhash %s", sealhash)
				return false
			}
		}
		// The submitted block is too old to accept, drop it.
		ethash.log.Warn("Work submitted is too old. height %d. sealhash %s. hash %s", solution.Header.Height, sealhash, solution.HeaderHash)
		return false
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case work := <-ethash.workCh:
			// Update current work with new received block.
			// Note same work can be past twice, happens when changing CPU threads.
			results = work.results

			makeWork(work.block)

			// Notify and requested URLs of the new work availability
			notifyWork()

		case work := <-ethash.fetchWorkCh:
			// Return current mining work to remote miner.
			if currentBlock == nil {
				work.errc <- errNoMiningWork
			} else {
				work.res <- currentWork
			}

		case result := <-ethash.submitWorkCh:
			// Verify submitted PoW solution based on maintained mining blocks.
			if submitWork(result.nonce, result.mixDigest, result.hash) {
				result.errc <- nil
			} else {
				result.errc <- errInvalidSealResult
			}

		case result := <-ethash.submitRateCh:
			// Trace remote sealer's hash rate by submitted value.
			rates[result.id] = hashrate{rate: result.rate, ping: time.Now()}
			close(result.done)

		case req := <-ethash.fetchRateCh:
			// Gather all hash rate submitted by remote sealer.
			var total uint64
			for _, rate := range rates {
				// this could overflow
				total += rate.rate
			}
			req <- total

		case <-ticker.C:
			// Clear stale submitted hash rate.
			for id, rate := range rates {
				if time.Since(rate.ping) > 10*time.Second {
					delete(rates, id)
				}
			}
			// Clear stale pending blocks
			if currentBlock != nil {
				for hash, block := range works {
					if block.Header.Height+staleThreshold <= currentBlock.Header.Height {
						delete(works, hash)
					}
				}
			}

		case errc := <-ethash.exitCh:
			// Exit remote loop if ethash is closed and return relevant error.
			errc <- nil
			ethash.log.Debug("Ethash remote sealer is exiting")
			return
		}
	}
}
