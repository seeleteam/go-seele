/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/keystore"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/rpc"
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
			name := flag.GetName()
			splitName := strings.Split(name, ",")

			var flagValue interface{}
			for _, n := range splitName {
				flagName := strings.TrimSpace(n)
				flagValue = context.Generic(flagName)
				if flagValue != nil {
					break
				}
			}

			args = append(args, flagValue)
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
		// Currently, flag is required to specify value.
		if c.NArg() > 0 {
			fmt.Printf("flag is not specified for value '%v'\n\n", c.Args().First())
			return cli.ShowCommandHelp(c, c.Command.Name)
		}

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
			return errInvalidSubcommand
		}

		printdata, arg, err := function(client)
		if err != nil {
			return err
		}

		find := 0
		if flags, ok := callFlags[namespace]; ok {
			if _, ok := flags[method]; ok {
				// use call method to get receipt
				find = 1
			}
		}

		if find == 1 {
			printdata, err = callTx(client, arg.(*types.Transaction))
			if err != nil {
				return err
			}

		} else {
			if err := sendTx(client, arg); err != nil {
				return err
			}
		}

		return resultHandler([]interface{}{}, printdata)
	}
}

func makeTransaction(context *cli.Context, client *rpc.Client) ([]interface{}, error) {
	key, txd, err := makeTransactionData(client)
	if err != nil {
		return nil, err
	}

	tx, err := util.GenerateTx(key.PrivateKey, txd.To, txd.Amount, txd.GasPrice, txd.GasLimit, txd.AccountNonce, txd.Payload)
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

	// print corresponding debt if exist
	debt := types.NewDebtWithoutContext(&tx)
	if debt != nil {
		fmt.Println()
		fmt.Println("It is a cross shard transaction, its debt is:")
		str, err := json.MarshalIndent(debt, "", "\t")
		if err != nil {
			return err
		}

		fmt.Println(string(str))
	}

	return nil
}

func getReceiptArgs(c *cli.Context, client *rpc.Client) ([]interface{}, error) {
	var abiJSON string
	if abiFile != "" {
		var err error
		if abiJSON, err = readABIFile(abiFile); err != nil {
			return nil, fmt.Errorf("failed to read abi file, err: %s", err)
		}
	}

	return []interface{}{hashValue, abiJSON}, nil
}
