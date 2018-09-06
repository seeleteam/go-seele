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
	"github.com/seeleteam/go-seele/rpc2"
	"github.com/urfave/cli"
)

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

func rpcAction(namespace string, method string) cli.ActionFunc {
	return rpcActionEx(namespace, method, parseCallArgs)
}

func rpcActionEx(namespace string, method string, argsFactory func(*cli.Context, *rpc.Client) ([]interface{}, error)) cli.ActionFunc {
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

		// print RPC call result in JSON format.
		if result != nil {
			encoded, err := json.MarshalIndent(result, "", "\t")
			if err != nil {
				return err
			}

			fmt.Println(string(encoded))
		}

		return nil
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
