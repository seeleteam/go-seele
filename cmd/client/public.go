/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/common/keystore"
	"github.com/seeleteam/go-seele/contract/system"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/rpc2"
	"github.com/seeleteam/go-seele/seele"
	"github.com/urfave/cli"
)

// RPCAction used to call rpc
func RPCAction(handler func(client *rpc.Client) (interface{}, error)) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		client, err := rpc.DialTCP(context.Background(), addressValue)
		if err != nil {
			return err
		}

		result, err := handler(client)
		if err != nil {
			return fmt.Errorf("get error when call rpc %s", err)
		}

		if result != nil {
			resultStr, err := json.MarshalIndent(result, "", "\t")
			if err != nil {
				return err
			}

			fmt.Printf("%s\n", resultStr)
		}

		return nil
	}
}

// GetInfoAction get current block info
func GetInfoAction(client *rpc.Client) (interface{}, error) {
	return util.GetInfo(client)
}

func getBalanceAction(client *rpc.Client) (interface{}, error) {
	account, err := MakeAddress(accountValue)
	if err != nil {
		return nil, err
	}

	var result seele.GetBalanceResponse
	err = client.Call(&result, "seele_getBalance", account)
	return result, err
}

// GetAccountNonceAction get current nonce
func GetAccountNonceAction(client *rpc.Client) (interface{}, error) {
	account, err := MakeAddress(accountValue)
	if err != nil {
		return nil, err
	}

	return util.GetAccountNonce(client, account)
}

// GetBlockHeightAction get block height
func GetBlockHeightAction(client *rpc.Client) (interface{}, error) {
	var result uint64
	err := client.Call(&result, "seele_getBlockHeight")
	return result, err
}

// GetBlockAction get block
func GetBlockAction(client *rpc.Client) (interface{}, error) {
	var result map[string]interface{}
	var err error

	if hashValue != "" {
		err = client.Call(&result, "seele_getBlockByHash", hashValue, fulltxValue)
	} else {
		err = client.Call(&result, "seele_getBlockByHeight", heightValue, fulltxValue)
	}

	return result, err
}

// GetLogsAction get logs
func GetLogsAction(client *rpc.Client) (interface{}, error) {
	var result []seele.GetLogsResponse
	err := client.Call(&result, "seele_getLogs", heightValue, contractValue, topicValue)

	return result, err
}

// callAction call transaction
func callAction(client *rpc.Client) (interface{}, error) {
	result := make(map[string]interface{})
	err := client.Call(&result, "seele_call", toValue, paloadValue, heightValue)

	return result, err
}

// AddTxAction send tx
func AddTxAction(client *rpc.Client) (interface{}, error) {
	tx, err := MakeTransaction(client)
	if err != nil {
		return nil, err
	}

	var result bool
	if err = client.Call(&result, "seele_addTx", *tx); err != nil || !result {
		fmt.Println("failed to send transaction")
		return nil, err
	}

	fmt.Println("transaction sent successfully")
	return tx, nil
}

// MakeAddress convert hex to address
func MakeAddress(value string) (common.Address, error) {
	if value == "" {
		return common.EmptyAddress, nil
	} else {
		return common.HexToAddress(value)
	}
}

// MakeTransaction generate transaction
func MakeTransaction(client *rpc.Client) (*types.Transaction, error) {
	pass, err := common.GetPassword()
	if err != nil {
		return nil, fmt.Errorf("failed to get password %s", err)
	}

	key, err := keystore.GetKey(fromValue, pass)
	if err != nil {
		return nil, fmt.Errorf("invalid sender key file. it should be a private key: %s", err)
	}

	txd, err := checkParameter(&key.PrivateKey.PublicKey, client)
	if err != nil {
		return nil, err
	}

	return util.GenerateTx(key.PrivateKey, txd.To, txd.Amount, txd.Fee, txd.AccountNonce, txd.Payload)
}

// HTLCTransaction generate HTLC transaction
func HTLCTransaction(client *rpc.Client) (*keystore.Key, *types.TransactionData, error) {
	pass, err := common.GetPassword()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get password %s", err)
	}

	key, err := keystore.GetKey(fromValue, pass)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid sender key file. it should be a private key: %s", err)
	}

	txd, err := checkParameter(&key.PrivateKey.PublicKey, client)
	if err != nil {
		return nil, nil, err
	}

	return key, txd, nil
}

// NewHTLC create HTLC
func NewHTLC(client *rpc.Client) (interface{}, error) {
	key, txd, err := HTLCTransaction(client)
	if err != nil {
		return nil, err
	}

	hashLockBytes, err := common.HexToHash(hashValue)
	if err != nil {
		return nil, fmt.Errorf("Failed to convert Hex to Hash %s", err)
	}

	timeLockNum, err := strconv.ParseInt(timeLockValue, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("Invalid timelock err %s, need int64", err)
	}

	var data system.HashTimeLock
	data.HashLock = hashLockBytes
	data.TimeLock = timeLockNum
	data.To = txd.To
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	input := make([]byte, len(dataBytes)+1)
	input[0] = system.CmdNewContract
	copy(input[1:], dataBytes)
	txd.Payload = input
	txd.To = system.HashTimeLockContractAddress
	tx, err := util.GenerateTx(key.PrivateKey, txd.To, txd.Amount, txd.Fee, txd.AccountNonce, txd.Payload)
	if err != nil {
		return nil, err
	}

	var result bool
	if err = client.Call(&result, "seele_addTx", *tx); err != nil || !result {
		return nil, errors.New("failed to send transaction")
	}

	output := make(map[string]interface{})
	output["Tx"] = *tx
	output["HashLock"] = hashValue
	output["TimeLock"] = timeLockValue
	return output, err
}

// Withdraw obtain seele from transaction
func Withdraw(client *rpc.Client) (interface{}, error) {
	key, txd, err := HTLCTransaction(client)
	if err != nil {
		return nil, err
	}

	txHashBytes, err := common.HexToHash(hashValue)
	if err != nil {
		return nil, fmt.Errorf("Failed to convert Hex to Hash %s", err)
	}

	preimageBytes, err := hexutil.HexToBytes(preimageValue)
	if err != nil {
		return nil, fmt.Errorf("Failed to convert Hex to Bytes %s", err)
	}

	var data system.Withdrawing
	data.Hash = txHashBytes
	data.Preimage = preimageBytes
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	txd.To = system.HashTimeLockContractAddress
	input := make([]byte, len(dataBytes)+1)
	input[0] = system.CmdWithdraw
	copy(input[1:], dataBytes)
	txd.Payload = input
	tx, err := util.GenerateTx(key.PrivateKey, txd.To, txd.Amount, txd.Fee, txd.AccountNonce, txd.Payload)
	if err != nil {
		return nil, err
	}

	var result bool
	if err = client.Call(&result, "seele_addTx", *tx); err != nil || !result {
		return nil, errors.New("failed to send transaction")
	}

	output := make(map[string]interface{})
	output["Tx"] = *tx
	output["hash"] = hashValue
	output["preimage"] = preimageValue
	return output, err
}

// Refund used to refund seele from HTLC
func Refund(client *rpc.Client) (interface{}, error) {
	key, txd, err := HTLCTransaction(client)
	if err != nil {
		return nil, err
	}

	txHashBytes, err := hexutil.HexToBytes(hashValue)
	if err != nil {
		return nil, fmt.Errorf("Failed to convert Hex to Bytes %s", err)
	}

	input := make([]byte, len(txHashBytes)+1)
	input[0] = system.CmdRefund
	copy(input[1:], txHashBytes)
	txd.To = system.HashTimeLockContractAddress
	txd.Payload = input
	tx, err := util.GenerateTx(key.PrivateKey, txd.To, txd.Amount, txd.Fee, txd.AccountNonce, txd.Payload)
	if err != nil {
		return nil, err
	}

	var result bool
	if err = client.Call(&result, "seele_addTx", *tx); err != nil || !result {
		return nil, errors.New("failed to send transaction")
	}

	output := make(map[string]interface{})
	output["Tx"] = *tx
	output["hash"] = hashValue
	return output, err
}

// GetContract used to get HTLC
func GetContract(client *rpc.Client) (interface{}, error) {
	key, txd, err := HTLCTransaction(client)
	if err != nil {
		return nil, err
	}

	txHashBytes, err := hexutil.HexToBytes(hashValue)
	if err != nil {
		return nil, fmt.Errorf("Failed to convert Hex to Bytes %s", err)
	}

	input := make([]byte, len(txHashBytes)+1)
	input[0] = system.CmdGetContract
	copy(input[1:], txHashBytes)
	txd.To = system.HashTimeLockContractAddress
	txd.Payload = input
	tx, err := util.GenerateTx(key.PrivateKey, txd.To, txd.Amount, txd.Fee, txd.AccountNonce, txd.Payload)
	if err != nil {
		return nil, err
	}

	var result bool
	if err = client.Call(&result, "seele_addTx", *tx); err != nil || !result {
		return nil, errors.New("failed to send transaction")
	}

	output := make(map[string]interface{})
	output["Tx"] = *tx
	output["hash"] = hashValue
	return output, err
}
