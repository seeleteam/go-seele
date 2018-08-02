/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/common/keystore"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/rpc2"
	"github.com/urfave/cli"
)

func GetAccountShardNumAction(c *cli.Context) error {
	var accountAddress common.Address
	if len(privateKeyValue) > 0 {
		key, err := crypto.LoadECDSAFromString(privateKeyValue)
		if err != nil {
			return fmt.Errorf("failed to load the private key: %s", err)
		}

		accountAddress = *(crypto.GetAddress(&key.PublicKey))
	} else {
		address, err := common.HexToAddress(accountValue)
		if err != nil {
			return fmt.Errorf("the account is invalid for: %v", err)
		}

		accountAddress = address
	}

	shard := accountAddress.Shard()
	fmt.Printf("shard number: %d\n", shard)
	return nil
}

func SaveKeyAction(c *cli.Context) error {
	privateKey, err := crypto.LoadECDSAFromString(privateKeyValue)
	if err != nil {
		return fmt.Errorf("invalid key: %s", err)
	}

	if fileNameValue == "" {
		return fmt.Errorf("invalid key file path")
	}

	pass, err := common.SetPassword()
	if err != nil {
		return fmt.Errorf("get password err %s", err)
	}

	key := keystore.Key{
		Address:    *crypto.GetAddress(&privateKey.PublicKey),
		PrivateKey: privateKey,
	}

	err = keystore.StoreKey(fileNameValue, pass, &key)
	if err != nil {
		return fmt.Errorf("failed to store the key file %s, %s", fileNameValue, err.Error())
	}

	fmt.Println("store key successful")
	return nil
}

func SignTxAction(c *cli.Context) error {
	var client *rpc.Client = nil
	if addressValue != "" {
		c, err := rpc.DialTCP(context.Background(), addressValue)
		if err != nil {
			return err
		}

		client = c
	}

	key, err := crypto.LoadECDSAFromString(privateKeyValue)
	if err != nil {
		return fmt.Errorf("failed to load key %s", err)
	}

	txd, err := checkParameter(&key.PublicKey, client)
	if err != nil {
		return err
	}

	var tx = types.Transaction{}
	tx.Data = *txd
	tx.Sign(key)

	result, err := json.MarshalIndent(tx, "", "\t")
	if err != nil {
		return err
	}

	fmt.Println(result)
	return nil
}

func GenerateKeyAction(c *cli.Context) error {
	var publicKey *common.Address
	var privateKey *ecdsa.PrivateKey
	var err error
	if shardValue > common.ShardCount {
		return fmt.Errorf("not supported shard number, shard number should be [0, %d]\n", common.ShardCount)
	} else if shardValue == 0 {
		publicKey, privateKey, err = crypto.GenerateKeyPair()
		if err != nil {
			return fmt.Errorf("failed to generate the key pair: %s", err)
		}
	} else {
		publicKey, privateKey = crypto.MustGenerateShardKeyPair(shardValue)
	}

	fmt.Printf("public key:  %s\n", publicKey.ToHex())
	fmt.Printf("private key: %s\n", hexutil.BytesToHex(crypto.FromECDSA(privateKey)))
	return nil
}
