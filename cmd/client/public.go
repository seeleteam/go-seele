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
	"github.com/urfave/cli"
)

type callArgsFactory func(*cli.Context, *rpc.Client) ([]interface{}, error)
type callResultHandler func(inputs []interface{}, result interface{}) error

func rpcFlags(callArgFlags ...cli.Flag) []cli.Flag {
	return append([]cli.Flag{addressFlag}, callArgFlags...)
}

func parseCallArgs(context *cli.Context, client *rpc.Client) ([]interface{}, error) {
	var args []interface{}

	for _, flag := range context.Command.Flags {
		if flag == addressFlag || flag == cli.HelpFlag {
			continue
		}

		if rf, ok := flag.(rpcFlag); ok {
			v, err := rf.getValue()
			if err != nil {
				return nil, err
			}

			args = append(args, v)
		} else {
			args = append(args, context.Generic(flag.GetName()))
		}
	}

	return args, nil
}

func handleCallResult(inputs []interface{}, result interface{}) error {
	if result == nil {
		return nil
	}

	if str, ok := result.(string); ok {
		fmt.Println(str)
		return nil
	}

	encoded, err := json.MarshalIndent(result, "", "\t")
	if err != nil {
		return err
	}

	fmt.Println(string(encoded))

	return nil
}

func rpcAction(namespace string, method string) cli.ActionFunc {
	return rpcActionEx(namespace, method, parseCallArgs, handleCallResult)
}

func rpcActionEx(namespace string, method string, argsFactory callArgsFactory, resultHandler callResultHandler) cli.ActionFunc {
	return func(c *cli.Context) error {
		client, err := rpc.DialTCP(context.Background(), addressValue)
		if err != nil {
			return err
		}

		args, err := argsFactory(c, client)
		if err != nil {
			return err
		}

		var result interface{}
		rpcMethod := fmt.Sprintf("%s_%s", namespace, method)
		if err = client.Call(&result, rpcMethod, args...); err != nil {
			return fmt.Errorf("Failed to call rpc, %s", err)
		}

		return resultHandler(args, result)
	}
}

func makeTransaction(context *cli.Context, client *rpc.Client) ([]interface{}, error) {
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

	tx, err := util.GenerateTx(key.PrivateKey, txd.To, txd.Amount, txd.Fee, txd.AccountNonce, txd.Payload)
	if err != nil {
		return nil, err
	}

	return []interface{}{*tx}, nil
}

func onTxAdded(inputs []interface{}, result interface{}) error {
	if !result.(bool) {
		fmt.Println("failed to send transaction")
	}

	tx := inputs[0].(types.Transaction)

	fmt.Println("transaction sent successfully")

	encoded, err := json.MarshalIndent(tx, "", "\t")
	if err != nil {
		return err
	}

	fmt.Println(string(encoded))

	return nil
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

// CreateHTLC create HTLC
func CreateHTLC(client *rpc.Client) (interface{}, error) {
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

// GetHTLC used to get HTLC
func GetHTLC(client *rpc.Client) (interface{}, error) {
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
