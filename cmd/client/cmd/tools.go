/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/seeleteam/go-seele/accounts/abi"
	"github.com/seeleteam/go-seele/accounts/abi/bind"
	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/common/keystore"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/urfave/cli"
)

// GetAccountShardNumAction is a action to get the shard number of account
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

// SaveKeyAction is a action to save the private key to the file
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

// SignTxAction is a action that signs a transaction
func SignTxAction(c *cli.Context) error {
	var client *rpc.Client
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

	fmt.Println(string(result))
	return nil
}

// GenerateKeyAction generate key by client command
func GenerateKeyAction(c *cli.Context) error {
	publicKey, privateKey, err := util.GenerateKey(shardValue)
	if err != nil {
		return err
	}

	fmt.Printf("public key:  %s\n", publicKey.Hex())
	fmt.Printf("private key: %s\n", hexutil.BytesToHex(crypto.FromECDSA(privateKey)))
	return nil
}

// DecryptKeyFileAction decrypt key file
func DecryptKeyFileAction(c *cli.Context) error {
	pass, err := common.GetPassword()
	if err != nil {
		return fmt.Errorf("failed to get password %s", err)
	}

	key, err := keystore.GetKey(fileNameValue, pass)
	if err != nil {
		return fmt.Errorf("invalid key file: %s", err)
	}

	fmt.Printf("public key:  %s\n", key.Address.Hex())
	fmt.Printf("private key: %s\n", hexutil.BytesToHex(crypto.FromECDSA(key.PrivateKey)))
	return nil
}

// GeneratePayloadAction is a action to generate the payload according to the abi string and method name and args
func GeneratePayloadAction(c *cli.Context) error {
	if abiFile == "" || methodName == "" {
		return fmt.Errorf("required flag(s) \"abi, method\" not set")
	}

	abiJSON, err := readABIFile(abiFile)
	if err != nil {
		return err
	}

	payload, err := generatePayload(abiJSON, methodName, c.StringSlice("args"))
	if err != nil {
		return fmt.Errorf("failed to parse the abi, err:%s", err)
	}

	fmt.Printf("payload: %s\n", hexutil.BytesToHex(payload))
	return nil
}

func generatePayload(abiStr, methodName string, args []string) ([]byte, error) {
	parsed, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		return nil, fmt.Errorf("failed to parse the abi, err:%s", err)
	}

	method, exist := parsed.Methods[methodName]
	if !exist {
		return nil, fmt.Errorf("method '%s' not found", methodName)
	}

	ss, err := bind.ParseArgs(method.Inputs, args)
	if err != nil {
		return nil, err
	}

	return parsed.Pack(methodName, ss...)
}

func readABIFile(abiFile string) (string, error) {
	if !common.FileOrFolderExists(abiFile) {
		return "", fmt.Errorf("The specified abi file[%s] does not exist", abiFile)
	}

	bytes, err := ioutil.ReadFile(abiFile)
	if err != nil {
		return "", fmt.Errorf("failed to read abi file, err: %s", err)
	}

	return string(bytes), nil
}

func GenerateTopicAction(c *cli.Context) error {
	if abiFile == "" || eventName == "" {
		return fmt.Errorf("required flag(s) \"abi, event\" not set")
	}

	abiJSON, err := readABIFile(abiFile)
	if err != nil {
		return err
	}

	topic, err := generateTopic(abiJSON, eventName)
	if err != nil {
		return fmt.Errorf("failed to parse the abi, err:%s", err)
	}

	fmt.Printf("event: %s, topic: %s\n", eventName, topic)
	return nil
}

func generateTopic(abiStr, eventName string) (string, error) {
	parsed, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		return "", fmt.Errorf("failed to parse the abi, err:%s", err)
	}

	event, exist := parsed.Events[eventName]
	if !exist {
		return "", fmt.Errorf("event '%s' not found", eventName)
	}

	return event.Id().Hex(), nil
}
