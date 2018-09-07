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

	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/common/keystore"
	"github.com/seeleteam/go-seele/contract/system"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/rpc2"
	"github.com/urfave/cli"
)

var (
	errInvalidCommand = errors.New("Faild to execute, invalid command")

	systemContract = map[string]map[string]func(client *rpc.Client) (interface{}, interface{}, error){
		"htlc": map[string]func(client *rpc.Client) (interface{}, interface{}, error){
			"create":   createHTLC,
			"withdraw": withdraw,
			"refund":   refund,
			"get":      getHTLC,
		},
	}
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

func rpcActionSystemContract(namespace string, method string, resultHandler callResultHandler) cli.ActionFunc {
	return func(c *cli.Context) error {
		client, err := rpc.DialTCP(context.Background(), addressValue)
		if err != nil {
			return err
		}

		functions, ok := systemContract[namespace]
		if !ok {
			return errInvalidCommand
		}

		function, ok := functions[method]
		if !ok {
			return errInvalidCommand
		}

		printdata, arg, err := function(client)
		if err != nil {
			return err
		}

		var result interface{}
		if err = client.Call(&result, "seele_addTx", arg); err != nil {
			return fmt.Errorf("Failed to call rpc, %s", err)
		}

		return resultHandler([]interface{}{}, printdata)
	}
}

func makeTransaction(context *cli.Context, client *rpc.Client) ([]interface{}, error) {
	key, txd, err := makeTransactionData(client)
	if err != nil {
		return nil, err
	}

	tx, err := util.GenerateTx(key.PrivateKey, txd.To, txd.Amount, txd.Fee, txd.AccountNonce, txd.Payload)
	if err != nil {
		return nil, err
	}

	return []interface{}{*tx}, nil
}

func makeTransactionData(client *rpc.Client) (*keystore.Key, *types.TransactionData, error) {
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

// CreateHTLC create HTLC
func createHTLC(client *rpc.Client) (interface{}, interface{}, error) {
	key, txd, err := makeTransactionData(client)
	if err != nil {
		return nil, nil, err
	}

	hashLockBytes, err := common.HexToHash(hashValue)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to convert Hex to Hash %s", err)
	}

	var data system.HashTimeLock
	data.HashLock = hashLockBytes
	data.TimeLock = timeLockValue
	data.To = txd.To
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, nil, err
	}

	txd.Payload = append([]byte{system.CmdNewContract}, dataBytes...)
	txd.To = system.HashTimeLockContractAddress
	tx, err := util.GenerateTx(key.PrivateKey, txd.To, txd.Amount, txd.Fee, txd.AccountNonce, txd.Payload)
	if err != nil {
		return nil, nil, err
	}

	output := make(map[string]interface{})
	output["Tx"] = *tx
	output["HashLock"] = hashValue
	output["TimeLock"] = timeLockValue
	return output, tx, err
}

// withdraw obtain seele from transaction
func withdraw(client *rpc.Client) (interface{}, interface{}, error) {
	key, txd, err := makeTransactionData(client)
	if err != nil {
		return nil, nil, err
	}

	txHashBytes, err := common.HexToHash(hashValue)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to convert Hex to Hash %s", err)
	}

	preimageBytes, err := hexutil.HexToBytes(preimageValue)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to convert Hex to Bytes %s", err)
	}

	var data system.Withdrawing
	data.Hash = txHashBytes
	data.Preimage = preimageBytes
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, nil, err
	}

	txd.To = system.HashTimeLockContractAddress
	txd.Payload = append([]byte{system.CmdWithdraw}, dataBytes...)
	tx, err := util.GenerateTx(key.PrivateKey, txd.To, txd.Amount, txd.Fee, txd.AccountNonce, txd.Payload)
	if err != nil {
		return nil, nil, err
	}

	output := make(map[string]interface{})
	output["Tx"] = *tx
	output["hash"] = hashValue
	output["preimage"] = preimageValue
	return output, tx, err
}

// refund used to refund seele from HTLC
func refund(client *rpc.Client) (interface{}, interface{}, error) {
	key, txd, err := makeTransactionData(client)
	if err != nil {
		return nil, nil, err
	}

	txHashBytes, err := hexutil.HexToBytes(hashValue)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to convert Hex to Bytes %s", err)
	}

	txd.To = system.HashTimeLockContractAddress
	txd.Payload = append([]byte{system.CmdRefund}, txHashBytes...)
	tx, err := util.GenerateTx(key.PrivateKey, txd.To, txd.Amount, txd.Fee, txd.AccountNonce, txd.Payload)
	if err != nil {
		return nil, nil, err
	}

	output := make(map[string]interface{})
	output["Tx"] = *tx
	output["hash"] = hashValue
	return output, tx, err
}

// getHTLC used to get HTLC
func getHTLC(client *rpc.Client) (interface{}, interface{}, error) {
	key, txd, err := makeTransactionData(client)
	if err != nil {
		return nil, nil, err
	}

	txHashBytes, err := hexutil.HexToBytes(hashValue)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to convert Hex to Bytes %s", err)
	}

	txd.To = system.HashTimeLockContractAddress
	txd.Payload = append([]byte{system.CmdGetContract}, txHashBytes...)
	tx, err := util.GenerateTx(key.PrivateKey, txd.To, txd.Amount, txd.Fee, txd.AccountNonce, txd.Payload)
	if err != nil {
		return nil, nil, err
	}

	output := make(map[string]interface{})
	output["Tx"] = *tx
	output["hash"] = hashValue
	return output, tx, err
}
