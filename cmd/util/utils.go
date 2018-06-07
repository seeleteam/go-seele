/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package util

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/seeleteam/go-seele/seele"
)

// Sendtx sends a transaction via RPC.
func Sendtx(client *rpc.Client, from *ecdsa.PrivateKey, to *common.Address, amount *big.Int, fee *big.Int, nonce uint64, payload []byte) bool {
	fromAddr := crypto.GetAddress(&from.PublicKey)

	var tx *types.Transaction
	var err error
	if to == nil {
		tx, err = types.NewContractTransaction(*fromAddr, amount, fee, nonce, payload)
	} else {
		switch to.Type() {
		case common.AddressTypeExternal:
			tx, err = types.NewTransaction(*fromAddr, *to, amount, fee, nonce)
		case common.AddressTypeContract:
			tx, err = types.NewMessageTransaction(*fromAddr, *to, amount, fee, nonce, payload)
		default:
			fmt.Println("unsupported address type", to.Type())
			return false
		}
	}

	if err != nil {
		fmt.Println("create transaction err ", err)
		return false
	}
	tx.Sign(from)

	var result bool
	err = client.Call("seele.AddTx", &tx, &result)
	if !result || err != nil {
		fmt.Printf("adding the tx failed: %s\n", err.Error())
		return false
	}

	fmt.Println("adding the tx succeeded.")
	printTx := seele.PrintableOutputTx(tx)
	str, err := json.MarshalIndent(printTx, "", "\t")
	if err != nil {
		fmt.Println("marshal transaction failed ", err)
		return true
	}

	fmt.Println(string(str))
	return true
}
