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

	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/common/keystore"
	"github.com/seeleteam/go-seele/core/types"
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
	return util.GetInfo(client)
}

func GetBalanceAction(client *rpc.Client) (interface{}, error) {
	account, err := MakeAddress(accountValue)
	if err != nil {
		return nil, err
	}

	return util.GetBalance(client, account)
}

func GetAccountNonceAction(client *rpc.Client) (interface{}, error) {
	account, err := MakeAddress(accountValue)
	if err != nil {
		return nil, err
	}

	return util.GetAccountNonce(client, account)
}

func GetBlockHeightAction(client *rpc.Client) (interface{}, error) {
	var result uint64
	err := client.Call(&result, "seele_getBlockHeight")
	return result, err
}

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

func GetLogsAction(client *rpc.Client) (interface{}, error) {
	var result []seele.GetLogsResponse
	err := client.Call(&result, "seele_getLogs", heightValue, contractValue, topicValue)

	return result, err
}

func CallAction(client *rpc.Client) (interface{}, error) {
	pass, err := common.GetPassword()
	if err != nil {
		return nil, fmt.Errorf("failed to get password: %s", err)
	}

	key, err := keystore.GetKey(fromValue, pass)
	if err != nil {
		return nil, fmt.Errorf("invalid sender key file. it should be a private key: %s", err)
	}

	contractAddr, err := common.HexToAddress(toValue)
	if err != nil {
		return nil, fmt.Errorf("invalid contract address: %s", err)
	}

	payload, err := hexutil.HexToBytes(paloadValue)
	if err != nil {
		return nil, fmt.Errorf("invalid payload, %s", err)
	}

	amount := big.NewInt(0)
	fee := big.NewInt(1)
	nonce := uint64(1)

	return util.Call(client, key.PrivateKey, &contractAddr, amount, fee, nonce, payload)
}

func AddTxAction(client *rpc.Client) (interface{}, error) {
	tx, err := MakeTransaction(client)
	if err != nil {
		return nil, err
	}

	var result bool
	err = client.Call(&result, "seele_addTx", tx)

	return result, err
}

func MakeAddress(value string) (common.Address, error) {
	if value == "" {
		return common.EmptyAddress, nil
	} else {
		return common.HexToAddress(value)
	}
}

func MakeTransaction(client *rpc.Client) (*types.Transaction, error) {
	pass, err := common.GetPassword()
	if err != nil {
		return nil, fmt.Errorf("failed to get password %s\n", err)
	}

	key, err := keystore.GetKey(fromValue, pass)
	if err != nil {
		return nil, fmt.Errorf("invalid sender key file. it should be a private key: %s\n", err)
	}

	txd, err := checkParameter(&key.PrivateKey.PublicKey, client)
	if err != nil {
		return nil, err
	}

	return util.GenerateTx(key.PrivateKey, txd.To, txd.Amount, txd.Fee, txd.AccountNonce, txd.Payload)
}
