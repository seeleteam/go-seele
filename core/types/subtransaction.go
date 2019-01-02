package types

import (
	"math/big"

	"fmt"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

const (
	UserDeposit        = iota // deposit
	StartUserExit             // start exit
	EndUserExit               // end exit
	ChallengedUserExit        // challenge result
)

// SubTransactionData subchain transaction data
type SubTransactionData struct {
	TxHash common.Hash // the hash of the executed transaction
	From   common.Address
	To     common.Address
	Nonce  uint64
	Amount *big.Int
	Bond   *big.Int
}

// SubTransaction SubTransaction class
type SubTransaction struct {
	Hash common.Hash // SubTransaction hash of SubTransactionData
	Data SubTransactionData
}

func (data *SubTransactionData) Hash() common.Hash {
	return crypto.MustHash(data)
}

func newSubTransaction(txHash common.Hash, args []interface{}, eventType int) (*SubTransaction, error) {
	data := SubTransactionData{
		TxHash: txHash,
	}

	var (
		ok     bool
		amount *big.Int
		bond   *big.Int
	)
	switch eventType {
	case UserDeposit:
		data.From = common.EmptyAddress

		if data.To, ok = args[0].(common.Address); !ok {
			return nil, fmt.Errorf("user deposit args[0] is not common.Address type")
		}

		if amount, ok = args[1].(*big.Int); !ok {
			return nil, fmt.Errorf("user deposit args[1] is not *big.Int type")
		}
		data.Amount = big.NewInt(0).Set(amount)
	case StartUserExit:
		if data.From, ok = args[0].(common.Address); !ok {
			return nil, fmt.Errorf("start user exit args[0] is not common.Address type")
		}

		data.To = common.EmptyAddress

		if amount, ok = args[1].(*big.Int); !ok {
			return nil, fmt.Errorf("start user exit args[1] is not *big.Int type")
		}
		data.Amount = big.NewInt(0).Set(amount)

		if bond, ok = args[2].(*big.Int); !ok {
			return nil, fmt.Errorf("start user exit args[2] is not *big.Int type")
		}
		data.Bond = bond

		if data.Nonce, ok = args[3].(uint64); !ok {
			return nil, fmt.Errorf("start user exit args[3] is not uint64 type")
		}
	case EndUserExit:
		if data.From, ok = args[0].(common.Address); !ok {
			return nil, fmt.Errorf("end user exit args[0] is not common.Address type")
		}

		data.To = common.EmptyAddress

		if amount, ok = args[1].(*big.Int); !ok {
			return nil, fmt.Errorf("end user exit args[1] is not *big.Int type")
		}
		data.Amount = big.NewInt(0).Set(amount)

		if data.Nonce, ok = args[2].(uint64); !ok {
			return nil, fmt.Errorf("end user exit args[2] is not uint64 type")
		}
	case ChallengedUserExit:
		if data.From, ok = args[0].(common.Address); !ok {
			return nil, fmt.Errorf("challenged user exit args[0] is not common.Address type")
		}

		if data.Nonce, ok = args[1].(uint64); !ok {
			return nil, fmt.Errorf("challenged user exit args[1] is not uint64 type")
		}

		if data.To, ok = args[2].(common.Address); !ok {
			return nil, fmt.Errorf("challenged user exit args[2] is not common.Address type")
		}
	}

	stx := &SubTransaction{
		Hash: data.Hash(),
		Data: data,
	}

	return stx, nil
}

func NewDepositSubTransaction(txHash common.Hash, args []interface{}) (*SubTransaction, error) {
	return newSubTransaction(txHash, args, UserDeposit)
}

func NewStartUserExitSubTransaction(txHash common.Hash, args []interface{}) (*SubTransaction, error) {
	return newSubTransaction(txHash, args, StartUserExit)
}

func NewEndUserExitSubTransaction(txHash common.Hash, args []interface{}) (*SubTransaction, error) {
	return newSubTransaction(txHash, args, EndUserExit)
}

func NewChallengedUserExitSubTransaction(txHash common.Hash, args []interface{}) (*SubTransaction, error) {
	return newSubTransaction(txHash, args, ChallengedUserExit)
}
