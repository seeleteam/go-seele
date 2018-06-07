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

func Sendtx(client *rpc.Client, from *ecdsa.PrivateKey, to common.Address, amount *big.Int, fee *big.Int, nonce uint64) bool {
	fromAddr := crypto.GetAddress(&from.PublicKey)

	tx, err := types.NewTransaction(*fromAddr, to, amount, fee, nonce)
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

func GetNonce(client *rpc.Client, address common.Address) uint64 {
	var nonce uint64
	err := client.Call("seele.GetAccountNonce", address, &nonce)
	if err != nil {
		fmt.Printf("getting the sender account nonce failed: %s\n", err.Error())
		return 0
	}

	fmt.Printf("got the sender account %s nonce: %d\n", address.ToHex(), nonce)

	return nonce
}
