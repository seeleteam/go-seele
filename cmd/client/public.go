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

func rpcFlags(callArgFlags ...cli.Flag) []cli.Flag {
	return append([]cli.Flag{addressFlag}, callArgFlags...)
}

func rpcAction(namespace string, method string) cli.ActionFunc {
	return func(c *cli.Context) error {
		// parse the call args from command flags.
		var args []interface{}
		for _, flag := range c.Command.Flags {
			if flag == addressFlag || flag == cli.HelpFlag {
				continue
			}

			if rf, ok := flag.(rpcFlag); ok {
				v, err := rf.getValue()
				if err != nil {
					return err
				}

				args = append(args, v)
			} else {
				args = append(args, c.Generic(flag.GetName()))
			}
		}

		// dail RPC connection.
		client, err := rpc.DialTCP(context.Background(), addressValue)
		if err != nil {
			return err
		}

		// RPC call
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

func getBlockAction(client *rpc.Client) (interface{}, error) {
	var result map[string]interface{}
	var err error

	if hashValue != "" {
		err = client.Call(&result, "seele_getBlockByHash", hashValue, fulltxValue)
	} else {
		err = client.Call(&result, "seele_getBlockByHeight", heightValue, fulltxValue)
	}

	return result, err
}

func addTxAction(client *rpc.Client) (interface{}, error) {
	tx, err := makeTransaction(client)
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

func makeTransaction(client *rpc.Client) (*types.Transaction, error) {
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
