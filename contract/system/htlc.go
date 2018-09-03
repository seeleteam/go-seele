package system

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
)

const (
	gasNewContract = uint64(100000)
	gasWithdraw    = uint64(5000)
	gasRefund      = uint64(5000)
	gasGetContract = uint64(5000)
)

const (
	cmdNewContract byte = iota
	cmdWithdraw
	cmdRefund
	cmdGetContract
)

var (
	htlcCommands = map[byte]*cmdInfo{
		cmdNewContract: &cmdInfo{gasNewContract, newHTLC},
		cmdWithdraw:    &cmdInfo{gasWithdraw, withdraw},
		cmdRefund:      &cmdInfo{gasRefund, refund},
		cmdGetContract: &cmdInfo{gasGetContract, getContract},
	}
)

var (
	errRedunedAgain            = errors.New("Failed to refund, owner have refunded")
	errRefundAfterWithdrawed   = errors.New("Failed to refund, receiver have withdrawed")
	errWithdrawAfterWithdrawed = errors.New("Failed to withdraw, receiver have withdrawed")
	errTimeLock                = errors.New("Failed to refund, time lock is not over")
	errTimeLockOver            = errors.New("Failed to withraw, time lock is over")
	errNotFutureTime           = errors.New("Failed to lock, time is not in future")
	errSender                  = errors.New("Failed to refund, only owner is allowed")
	errReceiver                = errors.New("Failed to withdraw, only receiver is allowed")
	errNilPointer              = errors.New("Failed to use nil pointer")
	errNoValueWithKey          = errors.New("Failed to get data with key")
	errHashNotMatch            = errors.New("Failed to use preimage to match hash")
)

type htlc struct {
	Tx *types.Transaction
	// HashLock is used to lock amount until provide preimage of hashlock
	Hashlock common.Hash
	// TimeLock is used to lock amount a period
	Timelock *big.Int
	// Refunded if refunded ture, otherwise false
	Refunded bool
	// Withdrawed if withdrawed true, otherwise false
	Withdrawed bool
	// Preimage is the hashlock preimage
	Preimage []byte
}

type hashTimeLock struct {
	// HashLock is used to lock amount until provide preimage of hashlock
	Hashlock common.Hash
	// TimeLock is used to lock amount a period
	Timelock *big.Int
}

type withdrawing struct {
	// Hash is the key of data
	Hash common.Hash
	// Preimage the hashlock preimage
	Preimage []byte
}

// create a HTLC to transfer value by hash-lock and time-lock
func newHTLC(lockbytes []byte, context *Context) ([]byte, error) {
	if context == nil {
		return nil, errNilPointer
	}

	var info hashTimeLock
	if err := json.Unmarshal(lockbytes, &info); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal lockbytes, %s", err)
	}

	if !validateAmount(context.tx) {
		return nil, errors.New("Failed to create HTLC, amount is less than or equal to 0")
	}

	if !isFutureTimeLock(info.Timelock, context.BlockHeader.CreateTimestamp) {
		return nil, errNotFutureTime
	}

	var data htlc
	data.Tx = context.tx
	data.Hashlock = info.Hashlock
	data.Timelock = info.Timelock
	data.Preimage = []byte{}
	value, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal data, %s", err)
	}

	subBalance(context.statedb, data.Tx.Data.From, data.Tx.Data.Amount)

	context.statedb.CreateAccount(hashTimeLockContractAddress)
	context.statedb.SetData(hashTimeLockContractAddress, data.Tx.Hash, value)

	return data.Tx.Hash.Bytes(), nil
}

// withdraw the seele from contract
func withdraw(jsonWithdraw []byte, context *Context) ([]byte, error) {
	if context == nil {
		return nil, errNilPointer
	}

	var input withdrawing
	if err := json.Unmarshal(jsonWithdraw, &input); err != nil {
		return nil, err
	}

	databytes, err := haveContract(context, input.Hash)
	if err != nil {
		return nil, err
	}

	var info htlc
	if err = json.Unmarshal(databytes, &info); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal data, %s", err)
	}

	if !hashlockMatches(info.Hashlock, input.Preimage) {
		return nil, errHashNotMatch
	}

	if err = withdrawable(&info, context.tx.Data.From, context); err != nil {
		return nil, err
	}

	info.Preimage = input.Preimage
	info.Withdrawed = true
	value, err := json.Marshal(info)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal data into json, %s", err)
	}

	context.statedb.SetData(hashTimeLockContractAddress, info.Tx.Hash, value)

	addBalance(context.statedb, context.tx.Data.From, info.Tx.Data.Amount)

	return value, nil
}

// refund the seele from contract after timelock
func refund(bytes []byte, context *Context) ([]byte, error) {
	if context == nil {
		return nil, errNilPointer
	}

	databytes, err := haveContract(context, common.BytesToHash(bytes))
	if err != nil {
		return nil, err
	}

	var info htlc
	if err := json.Unmarshal(databytes, &info); err != nil {
		return nil, err
	}

	if err = refundable(&info, context.tx.Data.From, context); err != nil {
		return nil, err
	}

	info.Refunded = true
	value, err := json.Marshal(info)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal data into json, %s", err)
	}

	context.statedb.SetData(hashTimeLockContractAddress, info.Tx.Hash, value)

	addBalance(context.statedb, context.tx.Data.From, info.Tx.Data.Amount)

	return value, nil
}

// getContract return contract info
func getContract(bytes []byte, context *Context) ([]byte, error) {
	if context == nil {
		return nil, errNilPointer
	}

	hash := common.BytesToHash(bytes)
	return haveContract(context, hash)
}

// get the data
func haveContract(context *Context, hash common.Hash) ([]byte, error) {
	bytes := context.statedb.GetData(hashTimeLockContractAddress, hash)
	if bytes == nil {
		return nil, errNoValueWithKey
	}

	return bytes, nil
}

// check if transfer amount is greater than 0
func validateAmount(tx *types.Transaction) bool {
	if tx.Data.Amount.Cmp(big.NewInt(0)) > 0 {
		return true
	}

	return false
}

// check timelock is futhure for now
func isFutureTimeLock(timelock, now *big.Int) bool {
	if now.Cmp(timelock) < 0 {
		return true
	}

	return false
}

// check if the preimage hash is equal to the hashlock
func hashlockMatches(hashlock common.Hash, preimage []byte) bool {
	hash := crypto.MustHash(preimage)
	if !hash.Equal(hashlock) {
		return false
	}

	return true
}

// check if withdraw is available
func withdrawable(data *htlc, receiver common.Address, context *Context) error {
	if !receiver.Equal(data.Tx.Data.To) {
		return errReceiver
	}

	if isFutureTimeLock(data.Timelock, context.BlockHeader.CreateTimestamp) != true {
		return errTimeLockOver
	}

	if data.Withdrawed {
		return errWithdrawAfterWithdrawed
	}

	return nil
}

// check if refund is available
func refundable(data *htlc, sender common.Address, context *Context) error {
	if !sender.Equal(data.Tx.Data.From) {
		return errSender
	}

	if isFutureTimeLock(data.Timelock, context.BlockHeader.CreateTimestamp) == true {
		return errTimeLock
	}

	if data.Withdrawed {
		return errRefundAfterWithdrawed
	}

	if data.Refunded {
		return errRedunedAgain
	}

	return nil
}

// add balance to account
func addBalance(s *state.Statedb, address common.Address, amount *big.Int) {
	s.AddBalance(address, amount)
}

// subBalance
func subBalance(s *state.Statedb, address common.Address, amount *big.Int) {
	s.SubBalance(address, amount)
}
