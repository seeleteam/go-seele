/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package util

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/rpc2"
	"github.com/seeleteam/go-seele/seele"
)

// GetAccountNonce get account nonce by account
func GetAccountNonce(client *rpc.Client, account common.Address) (uint64, error) {
	var nonce uint64
	err := client.Call(&nonce, "seele_getAccountNonce", account)
	return nonce, err
}

func GetInfo(client *rpc.Client) (seele.MinerInfo, error) {
	var info seele.MinerInfo
	err := client.Call(&info, "seele_getInfo")

	return info, err
}

func GenerateTx(from *ecdsa.PrivateKey, to common.Address, amount *big.Int, fee *big.Int, nonce uint64, payload []byte) (*types.Transaction, error) {
	fromAddr := crypto.GetAddress(&from.PublicKey)

	var tx *types.Transaction
	var err error
	if to.IsEmpty() {
		tx, err = types.NewContractTransaction(*fromAddr, amount, fee, nonce, payload)
	} else {
		switch to.Type() {
		case common.AddressTypeExternal:
			tx, err = types.NewTransaction(*fromAddr, to, amount, fee, nonce)
		case common.AddressTypeContract:
			tx, err = types.NewMessageTransaction(*fromAddr, to, amount, fee, nonce, payload)
		default:
			return nil, fmt.Errorf("unsupported address type: %d", to.Type())

		}
	}

	if err != nil {
		return nil, fmt.Errorf("create transaction err %s", err)
	}
	tx.Sign(from)

	return tx, nil
}

func GetTransactionByHash(client *rpc.Client, hash string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := client.Call(&result, "txpool_getTransactionByHash", hash)
	return result, err
}

func SendTx(client *rpc.Client, tx *types.Transaction) (bool, error) {
	var result bool
	err := client.Call(&result, "seele_addTx", *tx)

	return result, err
}
