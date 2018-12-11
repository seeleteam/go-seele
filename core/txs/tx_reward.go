/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package txs

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
)

var (
	emptyPayload = make([]byte, 0)
	emptySig     = crypto.Signature{Sig: make([]byte, 0)}
)

// NewRewardTx creates a reward transaction with the specified coinbase, reward and timestamp.
func NewRewardTx(coinbase common.Address, reward *big.Int, timestamp uint64) (*types.Transaction, error) {
	if err := validateReward(reward); err != nil {
		return nil, err
	}

	txData := types.TransactionData{
		Type:      types.TxTypeReward,
		From:      common.EmptyAddress,
		To:        coinbase,
		Amount:    new(big.Int).Set(reward),
		GasPrice:  common.Big0,
		Timestamp: timestamp,
		Payload:   emptyPayload,
	}

	tx := types.Transaction{
		Hash:      crypto.MustHash(txData),
		Data:      txData,
		Signature: emptySig,
	}

	return &tx, nil
}

func validateReward(reward *big.Int) error {
	if reward == nil {
		return types.ErrAmountNil
	}

	if reward.Sign() < 0 {
		return types.ErrAmountNegative
	}

	return nil
}
