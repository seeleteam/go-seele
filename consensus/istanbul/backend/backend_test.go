/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package backend

import (
	"bytes"
	"crypto/ecdsa"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus/istanbul"
	"github.com/seeleteam/go-seele/consensus/istanbul/validator"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
)

func TestSign(t *testing.T) {
	b := newBackend()
	data := []byte("Here is a string....")
	sig, err := b.Sign(data)
	if err != nil {
		t.Errorf("error mismatch: have %v, want nil", err)
	}
	//Check signature recover
	hashData := crypto.Keccak256([]byte(data))
	pubkey, _ := crypto.SigToPub(hashData, sig)
	signer := *crypto.GetAddress(pubkey)
	if signer != getAddress() {
		t.Errorf("address mismatch: have %v, want %s", signer.Hex(), getAddress().Hex())
	}
}

func TestCheckSignature(t *testing.T) {
	key := generatePrivateKey()
	data := []byte("Here is a string....")
	hashData := crypto.Keccak256([]byte(data))
	sig, _ := crypto.Sign(key, hashData)
	b := newBackend()
	a := getAddress()
	err := b.CheckSignature(data, a, sig.Sig)
	if err != nil {
		t.Errorf("error mismatch: have %v, want nil", err)
	}
	a = getInvalidAddress()
	err = b.CheckSignature(data, a, sig.Sig)
	if err != errInvalidSignature {
		t.Errorf("error mismatch: have %v, want %v", err, errInvalidSignature)
	}
}

func TestCheckValidatorSignature(t *testing.T) {
	vset, keys := newTestValidatorSet(5)

	// 1. Positive test: sign with validator's key should succeed
	data := []byte("dummy data")
	hashData := crypto.Keccak256([]byte(data))
	for i, k := range keys {
		// Sign
		sig, err := crypto.Sign(k, hashData)
		if err != nil {
			t.Errorf("error mismatch: have %v, want nil", err)
		}
		// CheckValidatorSignature should succeed
		addr, err := istanbul.CheckValidatorSignature(vset, data, sig.Sig)
		if err != nil {
			t.Errorf("error mismatch: have %v, want nil", err)
		}
		validator := vset.GetByIndex(uint64(i))
		if addr != validator.Address() {
			t.Errorf("validator address mismatch: have %v, want %v", addr, validator.Address())
		}
	}

	// 2. Negative test: sign with any key other than validator's key should return error
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Errorf("error mismatch: have %v, want nil", err)
	}
	// Sign
	sig, err := crypto.Sign(key, hashData)
	if err != nil {
		t.Errorf("error mismatch: have %v, want nil", err)
	}

	// CheckValidatorSignature should return ErrUnauthorizedAddress
	addr, err := istanbul.CheckValidatorSignature(vset, data, sig.Sig)
	if err != istanbul.ErrUnauthorizedAddress {
		t.Errorf("error mismatch: have %v, want %v", err, istanbul.ErrUnauthorizedAddress)
	}
	emptyAddr := common.Address{}
	if addr != emptyAddr {
		t.Errorf("address mismatch: have %v, want %v", addr, emptyAddr)
	}
}

func TestCommit(t *testing.T) {
	backend := newBackend()

	commitCh := make(chan *types.Block)
	// Case: it's a proposer, so the backend.commit will receive channel result from backend.Commit function
	testCases := []struct {
		expectedErr       error
		expectedSignature [][]byte
		expectedBlock     func() *types.Block
	}{
		{
			// normal case
			nil,
			[][]byte{append([]byte{1}, bytes.Repeat([]byte{0x00}, types.IstanbulExtraSeal-1)...)},
			func() *types.Block {
				chain, engine := newBlockChain(1)
				block := makeBlockWithoutSeal(chain, engine, chain.Genesis())
				expectedBlock, _ := engine.updateBlock(engine.chain.GetHeaderByHash(block.ParentHash()), block)
				return expectedBlock
			},
		},
		{
			// invalid signature
			errInvalidCommittedSeals,
			nil,
			func() *types.Block {
				chain, engine := newBlockChain(1)
				block := makeBlockWithoutSeal(chain, engine, chain.Genesis())
				expectedBlock, _ := engine.updateBlock(engine.chain.GetHeaderByHash(block.ParentHash()), block)
				return expectedBlock
			},
		},
	}

	for _, test := range testCases {
		expBlock := test.expectedBlock()
		go func() {
			select {
			case result := <-backend.commitCh:
				commitCh <- result
				return
			}
		}()

		backend.proposedBlockHash = expBlock.Hash()
		if err := backend.Commit(expBlock, test.expectedSignature); err != nil {
			if err != test.expectedErr {
				t.Errorf("error mismatch: have %v, want %v", err, test.expectedErr)
			}
		}

		if test.expectedErr == nil {
			// to avoid race condition is occurred by goroutine
			select {
			case result := <-commitCh:
				if result.Hash() != expBlock.Hash() {
					t.Errorf("hash mismatch: have %v, want %v", result.Hash(), expBlock.Hash())
				}
			case <-time.After(10 * time.Second):
				t.Fatal("timeout")
			}
		}
	}
}

func TestGetProposer(t *testing.T) {
	chain, engine := newBlockChain(1)
	block := makeBlock(chain, engine, chain.Genesis())
	err := chain.WriteBlock(block)
	if err != nil {
		panic(err)
	}

	expected := engine.GetProposer(1)
	actual := engine.Address()
	if actual != expected {
		t.Errorf("proposer mismatch: have %v, want %v", actual.Hex(), expected.Hex())
	}
}

/**
 * SimpleBackend
 * Private key: 0x6396d917f75c3f50bd845bd7fc32e9c1c73f7542b4dde25b6616adf41159a540
 * Public key: 0x04d85058c8d0b689f7b5ecc92728c4ed5dbdd60c01d3277dc6fe453761ce74dc78566c8088d22f8b05855fc26046f6b1e8cad03aa9f533953836ffa2e386567b98
 * Address: 0xb15bf0941562ee58256d1da3f6a6beffaa1b1791
 */
func getAddress() common.Address {
	return common.HexMustToAddres("0xb15bf0941562ee58256d1da3f6a6beffaa1b1791")
}

func getInvalidAddress() common.Address {
	return common.HexMustToAddres("0x99ea94bba74858c2ca43fec0050d3da8bc944141")
}

func generatePrivateKey() *ecdsa.PrivateKey {
	key := "0x6396d917f75c3f50bd845bd7fc32e9c1c73f7542b4dde25b6616adf41159a540"
	privateKey, err := crypto.LoadECDSAFromString(key)
	if err != nil {
		panic(err)
	}

	return privateKey
}

func newTestValidatorSet(n int) (istanbul.ValidatorSet, []*ecdsa.PrivateKey) {
	// generate validators
	keys := make(Keys, n)
	addrs := make([]common.Address, n)
	for i := 0; i < n; i++ {
		privateKey, _ := crypto.GenerateKey()
		keys[i] = privateKey
		addrs[i] = crypto.PubkeyToAddress(privateKey.PublicKey)
	}
	vset := validator.NewSet(addrs, istanbul.RoundRobin)
	sort.Sort(keys) //Keys need to be sorted by its public key address
	return vset, keys
}

type Keys []*ecdsa.PrivateKey

func (slice Keys) Len() int {
	return len(slice)
}

func (slice Keys) Less(i, j int) bool {
	return strings.Compare(crypto.PubkeyToAddress(slice[i].PublicKey).String(), crypto.PubkeyToAddress(slice[j].PublicKey).String()) < 0
}

func (slice Keys) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

func newBackend() (b *backend) {
	_, b = newBlockChain(4)
	key := generatePrivateKey()
	b.privateKey = key
	return
}
