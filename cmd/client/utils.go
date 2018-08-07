/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/rpc2"
)

const (
	// DefaultNonce is the default value of nonce,when you are not set the nonce flag in client sendtx command by --nonce .
	DefaultNonce uint64 = 0
)

func checkParameter(publicKey *ecdsa.PublicKey, client *rpc.Client) (*types.TransactionData, error) {
	info := &types.TransactionData{}
	var err error
	if len(toValue) > 0 {
		toAddr, err := common.HexToAddress(toValue)
		if err != nil {
			return info, fmt.Errorf("invalid receiver address: %s", err)
		}
		info.To = toAddr
	}

	amount, ok := big.NewInt(0).SetString(amountValue, 10)
	if !ok {
		return info, fmt.Errorf("invalid amount value")
	}
	info.Amount = amount

	fee, ok := big.NewInt(0).SetString(feeValue, 10)
	if !ok {
		return info, fmt.Errorf("invalid fee value")
	}
	info.Fee = fee

	fromAddr := crypto.GetAddress(publicKey)
	info.From = *fromAddr

	if client != nil {
		nonce, err := util.GetAccountNonce(client, *fromAddr)
		if err != nil {
			return info, fmt.Errorf("failed to get the sender account nonce: %s", err)
		}

		if nonceValue == nonce || nonceValue == DefaultNonce {
			info.AccountNonce = nonce
		} else {
			if nonceValue < nonce {
				return info, fmt.Errorf("your nonce is: %d,current nonce is: %d,you must set your nonce greater than or equal to current nonce", nonceValue, nonce)
			}
			info.AccountNonce = nonceValue
		}

		fmt.Printf("account %s current nonce: %d, sending nonce: %d\n", fromAddr.ToHex(), nonce, info.AccountNonce)
	} else {
		info.AccountNonce = nonceValue
	}

	payload := []byte(nil)
	if len(paloadValue) > 0 {
		if payload, err = hexutil.HexToBytes(paloadValue); err != nil {
			return info, fmt.Errorf("invalid payload, %s", err)
		}
	}
	info.Payload = payload

	return info, nil
}
