/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

 package spow

 import (
	 "math/big"
	 "runtime"
	 "testing"
	 "time"
	 "path/filepath"
 
	 "github.com/seeleteam/go-seele/common"
	 "github.com/seeleteam/go-seele/consensus"
	 "github.com/seeleteam/go-seele/core/types"
	 "github.com/seeleteam/go-seele/crypto"
	 "github.com/stretchr/testify/assert"
 )
 
 func Test_SetThreads(t *testing.T) {
	baseDir := common.GetTempFolder()
	datasetDir := filepath.Join(baseDir, "datasets")
	 engine := NewSpowEngine(1, datasetDir)
 
	 assert.Equal(t, engine.threads, 1)
 
	 engine.SetThreads(1)
	 assert.Equal(t, engine.threads, 1)
 
	 engine.SetThreads(2)
	 assert.Equal(t, engine.threads, 2)
 
	 engine.SetThreads(0)
	 assert.Equal(t, engine.threads, runtime.NumCPU())
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
		Witness:           common.StringToHash("Witness").Bytes(),
		SecondWitness:     common.StringToHash("Witness").Bytes(),
	}
}

func newTestBlockHeader2(t *testing.T) *types.BlockHeader {
	return &types.BlockHeader{
		PreviousBlockHash: common.StringToHash("PreviousBlockHash"),
		Creator:           randomAddress(t),
		StateHash:         common.StringToHash("StateHash"),
		TxHash:            common.StringToHash("TxHash"),
		Difficulty:        big.NewInt(2000000),
		Height:            1,
		CreateTimestamp:   big.NewInt(time.Now().Unix()),
		Witness:           common.StringToHash("Witness").Bytes(),
		SecondWitness:     common.StringToHash("SecondWitness").Bytes(),
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

func Test_verifyPair(t *testing.T) {
	header := newTestBlockHeader(t)
	err := verifyPair(header)
	assert.Equal(t, err, consensus.ErrBlockNonceInvalid)

	header = newTestBlockHeader2(t)
	err = verifyPair(header)
	assert.Equal(t, err, consensus.ErrBlockNonceInvalid)

}