/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package pow

import (
	"math/big"
	"runtime"
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

	engine.SetThreadNum(1)
	assert.Equal(t, engine.threads, 1)

	engine.SetThreadNum(2)
	assert.Equal(t, engine.threads, 2)

	engine.SetThreadNum(0)
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
