package system

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto/sha3"
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
		cmdNewContract: &cmdInfo{gasNewContract, newContract},
		cmdWithdraw:    &cmdInfo{gasWithdraw, withdraw},
		cmdRefund:      &cmdInfo{gasRefund, refund},
		cmdGetContract: &cmdInfo{gasGetContract, getContract},
	}
)

type htlc struct {
	Tx       *types.Transaction
	Hashlock string
	Timelock uint64
	Refund   bool
	Withdraw bool
	Preimage string
}

type lock struct {
	Hashlock string
	Timelock uint64
}

type withdrawing struct {
	Hash     common.Hash
	Preimage string
}

// create a contract to transfer value by hash-lock and time-lock
func newContract(lockbytes []byte, context *Context) ([]byte, error) {
	var info lock
	err := json.Unmarshal(lockbytes, &info)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal lockbytes err:%s\n", err)
	}

	err = fundsSend(context.tx)
	if err != nil {
		return nil, err
	}

	err = futureTimelock(info.Timelock)
	if err != nil {
		return nil, err
	}

	var data htlc
	data.Tx = context.tx
	data.Hashlock = info.Hashlock
	data.Timelock = info.Timelock
	data.Preimage = "0x0"
	data.Withdraw = false
	data.Refund = false

	value, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal data into json err:%s\n", err)
	}

	subBalance(context.statedb, data.Tx.Data.From, data.Tx.Data.Amount)

	context.statedb.CreateAccount(hashTimeLockContractAddress)
	context.statedb.SetData(hashTimeLockContractAddress, data.Tx.Hash, value)

	return data.Tx.Hash.Bytes(), nil
}

// withdraw the seele from contract
func withdraw(bytes []byte, context *Context) ([]byte, error) {
	var input withdrawing
	err := json.Unmarshal(bytes, &input)
	if err != nil {
		return nil, err
	}

	databytes, err := haveContract(context, input.Hash)
	if err != nil {
		return nil, err
	}

	var info htlc
	if unmarshal(databytes, &info) != nil {
		return nil, err
	}

	err = hashlockMatches(info.Hashlock, input.Preimage)
	if err != nil {
		return nil, err
	}

	err = withdrawable(&info, context.tx.Data.From)
	if err != nil {
		return nil, err
	}

	info.Preimage = input.Preimage
	info.Withdraw = true
	value, err := json.Marshal(info)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal data into json err:%s\n", err)
	}

	context.statedb.SetData(hashTimeLockContractAddress, info.Tx.Hash, value)

	addBalance(context.statedb, context.tx.Data.From, info.Tx.Data.Amount)

	return value, nil
}

// refund the seele from contract after timelock
func refund(bytes []byte, context *Context) ([]byte, error) {
	databytes, err := haveContract(context, common.BytesToHash(bytes))
	if err != nil {
		return nil, err
	}

	var info htlc
	if unmarshal(databytes, &info) != nil {
		return nil, err
	}

	err = refundable(&info, context.tx.Data.From)
	if err != nil {
		return nil, err
	}

	info.Refund = true
	value, err := json.Marshal(info)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal data into json err:%s\n", err)
	}

	context.statedb.SetData(hashTimeLockContractAddress, info.Tx.Hash, value)

	addBalance(context.statedb, context.tx.Data.From, info.Tx.Data.Amount)

	return value, nil
}

// getContract return contract info
func getContract(bytes []byte, context *Context) ([]byte, error) {
	hash := common.BytesToHash(bytes)
	return haveContract(context, hash)
}

// get the data
func haveContract(context *Context, hash common.Hash) ([]byte, error) {
	bytes := context.statedb.GetData(hashTimeLockContractAddress, hash)
	if bytes == nil {
		return nil, fmt.Errorf("Faild for no value with the key\n")
	}

	return bytes, nil
}

// check if transfer amount is greater than 0
func fundsSend(tx *types.Transaction) error {

	if tx.Data.Amount.Cmp(big.NewInt(0)) > 0 {
		return nil
	}

	return fmt.Errorf("Failed for amount less than or equal to 0\n")
}

// check timelock is futhure for now
func futureTimelock(timelock uint64) error {
	now := time.Now().Unix()
	if timelock > uint64(now) {
		return nil
	}

	return fmt.Errorf("Failed for timelock is not future for now\n")
}

// check if the preimage hash is equal to the hashlock
func hashlockMatches(hashlock string, preimage string) error {
	imagebytes, err := hexutil.HexToBytes(preimage)
	if err != nil {
		return err
	}

	hashbytes := sha3.Sum256(imagebytes)
	myHashLock := hexutil.BytesToHex(hashbytes[:])
	if hashlock != myHashLock {
		return fmt.Errorf("Failed to match the hashlock\n")
	}

	return nil
}

// check if withdraw is available
func withdrawable(data *htlc, receiver common.Address) error {
	if !bytes.Equal(data.Tx.Data.To[:], receiver[:]) {
		return fmt.Errorf("Failed for you is not the real receiver\n")
	}

	if futureTimelock(data.Timelock) != nil {
		return fmt.Errorf("Failed for timelock is passed\n")
	}

	if data.Withdraw {
		return fmt.Errorf("Failed for already withdrawed\n")
	}

	return nil
}

// check if refund is available
func refundable(data *htlc, sender common.Address) error {
	if !bytes.Equal(data.Tx.Data.From[:], sender[:]) {
		return fmt.Errorf("Failed for you is not the sender\n")
	}

	if futureTimelock(data.Timelock) == nil {
		return fmt.Errorf("Failed for timelock is not over\n")
	}

	if data.Withdraw {
		return fmt.Errorf("Failed for receiver have withdrawed\n")
	}

	if data.Refund {
		return fmt.Errorf("Failed for receiver have refunded\n")
	}

	return nil
}

// unmarshal htlc
func unmarshal(data []byte, value *htlc) error {
	return json.Unmarshal(data, value)
}

// add balance to account
func addBalance(s *state.Statedb, address common.Address, amount *big.Int) {
	s.AddBalance(address, amount)
}

// subBalance
func subBalance(s *state.Statedb, address common.Address, amount *big.Int) {
	s.SubBalance(address, amount)
}
