/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/rpc2"
	"github.com/seeleteam/go-seele/seele"
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

func GetInfoAction(client *rpc.Client) (interface{}, error) {
	var info seele.MinerInfo
	err := client.Call(&info, "seele_getInfo")

	return info, err
}

func GetBalanceAction(client *rpc.Client) (interface{}, error) {
	account, err := MakeAddress(accountValue)
	if err != nil {
		fmt.Println(err)
	}

	var result hexutil.Big
	err = client.Call(&result, "seele_getBalance", account)

	return (*big.Int)(&result), err
}

func MakeAddress(value string) (common.Address, error) {
	if value == "" {
		return common.EmptyAddress, nil
	} else {
		return common.HexToAddress(value)
	}
}
