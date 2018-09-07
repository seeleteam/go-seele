/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/keystore"
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
