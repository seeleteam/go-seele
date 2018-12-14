/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package txs

import (
	"fmt"
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
)

var (
	emptyPayload = make([]byte, 0)
	emptySig     = crypto.Signature{Sig: make([]byte, 0)}

	errEmptyToAddress    = errors.New("to address is empty")
	errCoinbaseMismatch  = errors.New("coinbase mismatch")
	errTimestampMismatch = errors.New("timestamp mismatch")
	errInvalidReward     = errors.New("invalid reward tx")
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

// ValidateRewardTx validates the specified reward tx.
func ValidateRewardTx(tx *types.Transaction, header *types.BlockHeader) error {
	if tx.Data.Type != types.TxTypeReward || !tx.Data.From.IsEmpty() || tx.Data.AccountNonce != 0 || tx.Data.GasPrice.Cmp(common.Big0) != 0 || tx.Data.GasLimit != 0 || len(tx.Data.Payload) != 0 {
		return errInvalidReward
	}

	// validate to address
	to := tx.Data.To
	if to.IsEmpty() {
		return errEmptyToAddress
	}

	if !to.Equal(header.Creator) {
		return errCoinbaseMismatch
	}

	// validate reward
	amount := tx.Data.Amount
	if err := validateReward(amount); err != nil {
		return err
	}

	reward := consensus.GetReward(header.Height)
	if reward == nil || reward.Cmp(amount) != 0 {
		return fmt.Errorf("invalid reward Amount, block height %d, want %s, got %s", header.Height, reward, amount)
	}

	// validate timestamp
	if tx.Data.Timestamp != header.CreateTimestamp.Uint64() {
		return errTimestampMismatch
	}

	return nil
}

// ApplyRewardTx applies the reward tx with specified statedb.
func ApplyRewardTx(tx *types.Transaction, statedb *state.Statedb) (*types.Receipt, error) {
	statedb.CreateAccount(tx.Data.To)
	statedb.AddBalance(tx.Data.To, tx.Data.Amount)

	hash, err := statedb.Hash()
	if err != nil {
		return nil, errors.NewStackedError(err, "failed to get statedb root hash")
	}

	receipt := &types.Receipt{
		TxHash:    tx.Hash,
		PostState: hash,
	}

	return receipt, nil
}
