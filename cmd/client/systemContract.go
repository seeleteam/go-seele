/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/contract/system"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/rpc2"
	"github.com/urfave/cli"
)

type handler func(client *rpc.Client) (interface{}, interface{}, error)

var (
	errInvalidCommand    = errors.New("invalid command")
	errInvalidSubcommand = errors.New("invalid subcommand")

	systemContract map[string]map[string]handler
)

func init() {
	systemContract = make(map[string]map[string]handler)

	systemContract["htlc"] = map[string]handler{
		"create":   createHTLC,
		"withdraw": withdraw,
		"refund":   refund,
		"get":      getHTLC,
	}

	systemContract["domain"] = map[string]handler{
		"register":     registerDomainName,
		"getregistrar": domainNameRegister,
	}
}

// createHTLC create HTLC
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

// generateHTLCKey
func generateHTLCKey(c *cli.Context) error {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret[:]); err != nil {
		return err
	}

	hash := crypto.MustHash(secret)
	fmt.Println("preimage:", hexutil.BytesToHex(secret[:]))
	fmt.Println("hash:", hash.ToHex())
	return nil
}

func registerDomainName(client *rpc.Client) (interface{}, interface{}, error) {
	return sendSystemContractTx(client, system.DomainNameContractAddress, system.CmdRegisterDomainName)
}

func domainNameRegister(client *rpc.Client) (interface{}, interface{}, error) {
	return sendSystemContractTx(client, system.DomainNameContractAddress, system.CmdDomainNameRegistrar)
}

func sendSystemContractTx(client *rpc.Client, to common.Address, method byte) (output types.Transaction, tx *types.Transaction, err error) {
	key, txd, err := makeTransactionData(client)
	if err != nil {
		return
	}

	txd.To = to
	txd.Payload = append([]byte{method}, txd.Payload...)
	tx, err = util.GenerateTx(key.PrivateKey, txd.To, txd.Amount, txd.Fee, txd.AccountNonce, txd.Payload)
	if err != nil {
		return
	}

	output = *tx
	return
}
