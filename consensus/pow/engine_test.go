/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package pow

import (
	"math/big"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/stretchr/testify/assert"
)

func Test_SetThreads(t *testing.T) {
	engine := NewEngine(1)

	assert.Equal(t, engine.threads, 1)

	engine.SetThreads(1)
	assert.Equal(t, engine.threads, 1)

	engine.SetThreads(2)
	assert.Equal(t, engine.threads, 2)

	engine.SetThreads(0)
	assert.Equal(t, engine.threads, runtime.NumCPU())
}

func Test_VerifyTarget(t *testing.T) {
	// block is validated for difficulty is so low
	header := newTestBlockHeader(t)
	err := verifyTarget(header)
	assert.Equal(t, err, nil)

	// block is not validated for difficulty is so high
	header.Difficulty = big.NewInt(10000000000)
	err = verifyTarget(header)
	assert.Equal(t, err, consensus.ErrBlockNonceInvalid)
}

func newTestBlockHeader(t *testing.T) *types.BlockHeader {
	return &types.BlockHeader{
		PreviousBlockHash: common.StringToHash("PreviousBlockHash"),
		Creator:           randomAddress(t),
		StateHash:         common.StringToHash("StateHash"),
		TxHash:            common.StringToHash("TxHash"),
		Difficulty:        big.NewInt(1),
		Height:            1,
		CreateTimestamp:   big.NewInt(time.Now().Unix()),
	}
}

func randomAddress(t *testing.T) common.Address {
	privKey, keyErr := crypto.GenerateKey()
	if keyErr != nil {
		t.Fatalf("Failed to generate ECDSA private key, error = %s", keyErr.Error())
	}
	hexAddress := crypto.PubkeyToString(&privKey.PublicKey)

	return common.HexMustToAddres(hexAddress)
}

func Test_Seal(t *testing.T) {
	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go sealTest(t, &wg)
	}

	wg.Wait()
}

func sealTest(t *testing.T, wg *sync.WaitGroup) {
	engine := NewEngine(10)

	stop := make(chan struct{})
	results := make(chan *types.Block)

	header := newTestBlockHeader(t)
	header.Difficulty = big.NewInt(900)

	block := &types.Block{
		Header: header,
	}

	go func() {
		defer wg.Done()

		timer := time.NewTimer(5 * time.Second)
		count := 0
	test:
		for {
			select {
			case b := <-results:
				if b != nil {
					count++
					if count > 1 {
						t.Fatalf("got too many block, %d", count)
					}
				}
			case <-timer.C:
				break test
			}
		}
	}()

	engine.Seal(nil, block, stop, results)
}
