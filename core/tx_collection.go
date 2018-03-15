/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"sort"

	"github.com/seeleteam/go-seele/core/types"
)

type txCollection struct {
	nonceToTxMap map[uint64]*types.Transaction
}

func newTxCollection() *txCollection {
	return &txCollection{
		nonceToTxMap: make(map[uint64]*types.Transaction),
	}
}

func (collection *txCollection) add(tx *types.Transaction) {
	collection.nonceToTxMap[tx.Data.AccountNonce] = tx
}

func (collection *txCollection) getTxs() []*types.Transaction {
	txs := make([]*types.Transaction, 0, len(collection.nonceToTxMap))

	for _, tx := range collection.nonceToTxMap {
		txs = append(txs, tx)
	}

	return txs
}

func (collection *txCollection) getTxsOrderByNonceAsc() []*types.Transaction {
	txs := collection.getTxs()

	sort.Slice(txs, func(i, j int) bool {
		return txs[i].Data.AccountNonce < txs[j].Data.AccountNonce
	})

	return txs
}
