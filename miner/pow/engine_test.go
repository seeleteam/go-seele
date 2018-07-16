/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package pow

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
)

func Test_GetDifficult(t *testing.T) {
	diff1 := big.NewInt(10000)
	diff2 := getDiff(5, diff1)
	assert.Equal(t, diff2.Int64(), int64(10004))

	diff3 := getDiff(16, diff1)
	assert.Equal(t, diff3.Int64(), int64(10000))

	diff4 := getDiff(24, diff1)
	assert.Equal(t, diff4.Int64(), int64(9996))

	diff5 := getDiff(11000, diff1)
	assert.Equal(t, diff5.Int64(), int64(9604))

	diff6 := getDiff(100, diff1)
	assert.Equal(t, diff6.Int64(), int64(9964))
}

func getDiff(interval uint64, diff *big.Int) *big.Int {
	header := &types.BlockHeader{
		CreateTimestamp: big.NewInt(0),
		Difficulty:      diff,
		Height:          10,
	}

	return GetDifficult(interval, header)
}

func Test_ValidateRewardAmount(t *testing.T) {
	var engine Engine
	var height uint64

	// block height and reward amount is equal
	err := engine.ValidateRewardAmount(height, GetReward(height))
	assert.Equal(t, err, nil)

	// block height and reward amount is equal
	height = blockNumberPerEra
	err = engine.ValidateRewardAmount(height, GetReward(height))
	assert.Equal(t, err, nil)

	// block height and reward amount is not equal
	height = blockNumberPerEra * 2
	err = engine.ValidateRewardAmount(height, GetReward(blockNumberPerEra))
	assert.Equal(t, err.Error(), fmt.Sprintf("invalid reward amount, block height %d, want %s, got %s", height, GetReward(height), GetReward(blockNumberPerEra)))
}

func Test_ValidateHeader(t *testing.T) {
	var engine Engine

	// block is validated for difficulty is so low
	header := newTestBlockHeader(t)
	err := engine.ValidateHeader(header)
	assert.Equal(t, err, nil)

	// block is not validated for difficulty is so high
	header.Difficulty = big.NewInt(10000000000)
	err = engine.ValidateHeader(header)
	assert.Equal(t, err, errBlockNonceInvalid)
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
		Nonce:             1,
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
