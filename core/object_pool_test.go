/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package core

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database/leveldb"
)

func randomAccount(t *testing.T) (*ecdsa.PrivateKey, common.Address) {
	privKey, keyErr := crypto.GenerateKey()
	if keyErr != nil {
		t.Fatalf("Failed to generate ECDSA private key, error = %s", keyErr.Error())
	}

	hexAddress := crypto.PubkeyToString(&privKey.PublicKey)

	return privKey, common.HexMustToAddres(hexAddress)
}

func newTestPoolTx(t *testing.T, amount int64, nonce uint64) *poolItem {
	return newTestPoolTxWithNonce(t, amount, nonce, 1)
}

func newTestPoolTxWithNonce(t *testing.T, amount int64, nonce uint64, price int64) *poolItem {
	fromPrivKey, fromAddress := randomAccount(t)

	return newTestPoolEx(t, fromPrivKey, fromAddress, amount, nonce, price)
}

func newTestPoolEx(t *testing.T, fromPrivKey *ecdsa.PrivateKey, fromAddress common.Address, amount int64, nonce uint64, price int64) *poolItem {
	_, toAddress := randomAccount(t)

	tx, _ := types.NewTransaction(fromAddress, toAddress, big.NewInt(amount), big.NewInt(price), nonce)
	tx.Sign(fromPrivKey)

	return newPooledItem(tx)
}

type mockBlockchain struct {
	statedb    *state.Statedb
	chainStore store.BlockchainStore
	dispose    func()
}

func newMockBlockchain() *mockBlockchain {
	statedb, err := state.NewStatedb(common.EmptyHash, nil)
	if err != nil {
		panic(err)
	}

	db, dispose := leveldb.NewTestDatabase()
	chainStore := store.NewBlockchainDatabase(db)
	return &mockBlockchain{statedb, chainStore, dispose}
}

func (chain mockBlockchain) GetCurrentState() (*state.Statedb, error) {
	return chain.statedb, nil
}

func (chain mockBlockchain) GetStore() store.BlockchainStore {
	return chain.chainStore
}
