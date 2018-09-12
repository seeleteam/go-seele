/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/contract/system"
	"github.com/seeleteam/go-seele/rpc2"
	"github.com/urfave/cli"
)

// createHTLC create HTLC
func createHTLC(client *rpc.Client) (interface{}, interface{}, error) {
	key, txd, err := makeTransactionData(client)
	if err != nil {
		return nil, nil, err
	}

	hashLockBytes, err := hexutil.HexToBytes(hashValue)
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
	amountValue = "0"
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
	amountValue = "0"
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
	amountValue = "0"
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

// generateHTLCKey generate HTLC preimage and preimage hash
func generateHTLCKey(c *cli.Context) error {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret[:]); err != nil {
		return err
	}

	hash := system.Sha256Hash(secret)
	fmt.Println("preimage:", hexutil.BytesToHex(secret[:]))
	fmt.Println("hash:", hexutil.BytesToHex(hash[:]))
	return nil
}

// generateHTLCTime generate HTLC time lock
func generateHTLCTime(c *cli.Context) error {
	locktime := time.Now().Unix() + timeLockValue
	fmt.Println("locktime:", locktime)
	return nil
}

// decodeHTLC decode htlc information
func decodeHTLC(c *cli.Context) error {
	result, err := system.DecodeHTLC(payloadValue)
	if err != nil {
		return err
	}

	return handleCallResult(nil, result)
}
