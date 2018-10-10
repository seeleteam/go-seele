/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/contract/system"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/urfave/cli"
)

// createHTLC create HTLC
func createHTLC(client *rpc.Client) (interface{}, interface{}, error) {
	hashLockBytes, err := hexutil.HexToBytes(hashValue)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to convert Hex to Hash %s", err)
	}

	var data system.HashTimeLock
	data.HashLock = hashLockBytes
	data.TimeLock = timeLockValue
	toAddr, err := common.HexToAddress(toValue)
	if err != nil {
		return nil, nil, err
	}

	data.To = toAddr
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, nil, err
	}

	tx, err := sendSystemContractTx(client, system.HashTimeLockContractAddress, system.CmdNewContract, dataBytes)
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

	tx, err := sendSystemContractTx(client, system.HashTimeLockContractAddress, system.CmdWithdraw, dataBytes)
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
	txHashBytes, err := hexutil.HexToBytes(hashValue)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to convert Hex to Bytes %s", err)
	}

	tx, err := sendSystemContractTx(client, system.HashTimeLockContractAddress, system.CmdRefund, txHashBytes)
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
	priceValue = "1"
	txHashBytes, err := hexutil.HexToBytes(hashValue)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to convert Hex to Bytes %s", err)
	}

	tx, err := sendSystemContractTx(client, system.HashTimeLockContractAddress, system.CmdGetContract, txHashBytes)
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
