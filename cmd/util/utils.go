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
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/seeleteam/go-seele/seele"
)

const (
	// DefaultNonce is the default value of nonce,when you are not set the nonce flag in client sendtx command by --nonce .
	DefaultNonce uint64 = 0
)

type TxInfo struct {
	Amount  *string // amount specifies the coin amount to be transferred
	To      *string // to is the public address of the receiver
	From    *string // from is the key file path of the sender
	Fee     *string // transaction fee
	Payload *string // transaction payload in hex format
	Nonce   *uint64 // nonce number in transaction or contract
}

// Call call a transaction via RPC.
func Call(client *rpc.Client, from *ecdsa.PrivateKey, to *common.Address, amount *big.Int, fee *big.Int, nonce uint64, payload []byte) (*map[string]interface{}, bool) {
	fromAddr := crypto.GetAddress(&from.PublicKey)

	var tx *types.Transaction
	var err error
	switch to.Type() {
	case common.AddressTypeContract:
		tx, err = types.NewMessageTransaction(*fromAddr, *to, amount, fee, nonce, payload)
	default:
		fmt.Println("unsupported address type:", to.Type())
		return nil, false
	}

	if err != nil {
		fmt.Println("failed to create transaction:", err)
		return nil, false
	}
	tx.Sign(from)

	request := seele.CallRequest{
		Tx:     tx,
		Height: -1,
	}
	result := make(map[string]interface{})
	if err = client.Call("seele.Call", &request, &result); err != nil {
		fmt.Println("failed to call contract:", err.Error())
		return nil, false
	}

	return &result, true
}

// Sendtx sends a transaction via RPC.
func Sendtx(client *rpc.Client, from *ecdsa.PrivateKey, to *common.Address, amount *big.Int, fee *big.Int, nonce uint64, payload []byte) (*types.Transaction, bool) {
	fromAddr := crypto.GetAddress(&from.PublicKey)

	var tx *types.Transaction
	var err error
	if to == nil || to.IsEmpty() {
		tx, err = types.NewContractTransaction(*fromAddr, amount, fee, nonce, payload)
	} else {
		switch to.Type() {
		case common.AddressTypeExternal:
			tx, err = types.NewTransaction(*fromAddr, *to, amount, fee, nonce)
		case common.AddressTypeContract:
			tx, err = types.NewMessageTransaction(*fromAddr, *to, amount, fee, nonce, payload)
		default:
			fmt.Println("unsupported address type:", to.Type())
			return nil, false
		}
	}

	if err != nil {
		fmt.Println("create transaction err ", err)
		return nil, false
	}
	tx.Sign(from)

	var result bool
	err = client.Call("seele.AddTx", &tx, &result)
	if !result || err != nil {
		fmt.Printf("adding the tx failed: %s\n", err.Error())
		return nil, false
	}

	return tx, true
}

func GetNonce(client *rpc.Client, address common.Address) uint64 {
	var nonce uint64
	err := client.Call("seele.GetAccountNonce", address, &nonce)
	if err != nil {
		fmt.Printf("getting the sender account nonce failed: %s\n", err.Error())
		return 0
	}

	fmt.Printf("got account: %s nonce: %d\n", address.ToHex(), nonce)

	return nonce
}

func CheckParameter(parameter TxInfo, publicKey *ecdsa.PublicKey, client *rpc.Client) (*types.TransactionData, bool) {
	info := &types.TransactionData{}
	var err error
	if len(*parameter.To) > 0 {
		toAddr := new(common.Address)
		if *toAddr, err = common.HexToAddress(*parameter.To); err != nil {
			fmt.Printf("invalid receiver address: %s\n", err.Error())
			return info, false
		}
		info.To = *toAddr
	}

	amount, ok := big.NewInt(0).SetString(*parameter.Amount, 10)
	if !ok {
		fmt.Println("invalid amount value")
		return info, false
	}
	info.Amount = amount

	fee, ok := big.NewInt(0).SetString(*parameter.Fee, 10)
	if !ok {
		fmt.Println("invalid fee value")
		return info, false
	}
	info.Fee = fee

	fromAddr := crypto.GetAddress(publicKey)
	info.From = *fromAddr

	var nonce uint64
	err = client.Call("seele.GetAccountNonce", fromAddr, &nonce)
	if err != nil {
		fmt.Printf("getting the sender account nonce failed: %s\n", err.Error())
		return info, false
	}
	if *parameter.Nonce == nonce || *parameter.Nonce == DefaultNonce {
		info.AccountNonce = nonce
	} else {
		if *parameter.Nonce < nonce {
			fmt.Printf("your nonce is: %d,current nonce is: %d,you must set your nonce greater than or equal to current nonce\n", *parameter.Nonce, nonce)
			return info, false
		}
		info.AccountNonce = *parameter.Nonce
	}
	fmt.Printf(" the sender account %s current nonce: %d,sending nonce: %d\n", fromAddr.ToHex(), nonce, info.AccountNonce)

	payload := []byte(nil)
	if len(*parameter.Payload) > 0 {
		if payload, err = hexutil.HexToBytes(*parameter.Payload); err != nil {
			fmt.Println("invalid payload,", err.Error())
			return info, false
		}
	}
	info.Payload = payload
	return info, true
}
