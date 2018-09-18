/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"errors"

	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/rpc"
)

type handler func(client *rpc.Client) (interface{}, interface{}, error)

var (
	errInvalidCommand    = errors.New("invalid command")
	errInvalidSubcommand = errors.New("invalid subcommand")

	systemContract = map[string]map[string]handler{
		"htlc": map[string]handler{
			"create":   createHTLC,
			"withdraw": withdraw,
			"refund":   refund,
			"get":      getHTLC,
		},
		"domain": map[string]handler{
			"create":   createDomainName,
			"getOwner": getDomainNameOwner,
		},
		"subchain": map[string]handler{
			"register": registerSubChain,
			"query":    querySubChain,
		},
	}
)

// sendSystemContractTx send system contract transaction
func sendSystemContractTx(client *rpc.Client, to common.Address, method byte, payload []byte) (*types.Transaction, error) {
	key, txd, err := makeTransactionData(client)
	if err != nil {
		return nil, err
	}

	txd.To = to
	txd.Payload = append([]byte{method}, payload...)
	tx, err := util.GenerateTx(key.PrivateKey, txd.To, txd.Amount, txd.Fee, txd.AccountNonce, txd.Payload)
	if err != nil {
		return nil, err
	}

	return tx, err
}
