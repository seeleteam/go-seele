/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package backend

import (
	"bytes"
	"crypto/ecdsa"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/consensus/istanbul"
	"github.com/seeleteam/go-seele/consensus/pow"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database/leveldb"
)

var genesisAccount = crypto.MustGenerateShardAddress(1)

func newTestGenesis(n int) (*core.Genesis, []*ecdsa.PrivateKey) {
	accounts := map[common.Address]*big.Int{
		*genesisAccount: new(big.Int).Mul(big.NewInt(4), common.SeeleToFan),
	}

	// Setup validators
	var nodeKeys = make([]*ecdsa.PrivateKey, n)
	var addrs = make([]common.Address, n)
	for i := 0; i < n; i++ {
		nodeKeys[i], _ = crypto.GenerateKey()
		addrs[i] = crypto.PubkeyToAddress(nodeKeys[i].PublicKey)
	}

	genesis := core.GetGenesis(core.NewGenesisInfo(accounts, 1, 1, big.NewInt(0), types.IstanbulConsensus, addrs))
	return genesis, nodeKeys
}

// in this test, we can set n to 1, and it means we can process Istanbul and commit a
// block by one node. Otherwise, if n is larger than 1, we have to generate
// other fake events to process Istanbul.
func newBlockChain(n int) (*core.Blockchain, *backend) {
	db, _ := leveldb.NewTestDatabase()
	bcStore := store.NewCachedStore(store.NewBlockchainDatabase(db))

	genesis, nodeKeys := newTestGenesis(n)
	if err := genesis.InitializeAndValidate(bcStore, db); err != nil {
		panic(err)
	}

	bc, err := core.NewBlockchain(bcStore, db, "", pow.NewEngine(1), nil)
	if err != nil {
		panic(err)
	}

	config := istanbul.DefaultConfig
	b, _ := New(config, nodeKeys[0], db).(*backend)
	b.Start(bc, bc.CurrentBlock, func(hash common.Hash) bool {
		return false
	})

	snap, err := b.snapshot(bc, 0, common.Hash{}, nil)
	if err != nil {
		panic(err)
	}
	if snap == nil {
		panic("failed to get snapshot")
	}
	proposerAddr := snap.ValSet.GetProposer().Address()

	// find proposer key
	for _, key := range nodeKeys {
		addr := crypto.PubkeyToAddress(key.PublicKey)
		if addr.String() == proposerAddr.String() {
			b.privateKey = key
			b.address = addr
		}
	}

	return bc, b
}

func makeHeader(parent *types.Block, config *istanbul.Config) *types.BlockHeader {
	header := &types.BlockHeader{
		PreviousBlockHash: parent.Hash(),
		Height:            parent.Height() + 1,
		CreateTimestamp:   new(big.Int).Add(parent.Header.CreateTimestamp, new(big.Int).SetUint64(config.BlockPeriod)),
		Difficulty:        defaultDifficulty,
		ExtraData:         parent.Header.ExtraData,
		StateHash:         parent.Header.StateHash,
	}
	return header
}

func makeBlock(chain *core.Blockchain, engine *backend, parent *types.Block) *types.Block {
	block := makeBlockWithoutSeal(chain, engine, parent)
	block, err := engine.SealWithReturn(chain, block, nil)
	if err != nil {
		panic(err)
	}

	return block
}

func makeBlockWithoutSeal(chain *core.Blockchain, engine *backend, parent *types.Block) *types.Block {
	header := makeHeader(parent, engine.config)
	engine.Prepare(chain, header)
	reward := consensus.GetReward(header.Height)
	rewardTx, err := types.NewRewardTransaction(header.Creator, reward, header.CreateTimestamp.Uint64())
	if err != nil {
		panic(err)
	}

	state, err := chain.GetState(parent.Header.StateHash)
	if err != nil {
		panic(err)
	}

	state.AddBalance(header.Creator, reward)
	header.StateHash, err = state.Hash()
	if err != nil {
		panic(err)
	}

	block := types.NewBlock(header, []*types.Transaction{rewardTx}, nil, nil)
	return block
}

func TestPrepare(t *testing.T) {
	chain, engine := newBlockChain(1)
	header := makeHeader(chain.Genesis(), engine.config)
	err := engine.Prepare(chain, header)
	if err != nil {
		t.Errorf("error mismatch: have %v, want nil", err)
	}
	header.PreviousBlockHash = common.StringToHash("1234567890")
	err = engine.Prepare(chain, header)
	if err != consensus.ErrBlockInvalidParentHash {
		t.Errorf("error mismatch: have %v, want %v", err, consensus.ErrBlockInvalidParentHash)
	}
}

func TestSealStopChannel(t *testing.T) {
	chain, engine := newBlockChain(4)
	block := makeBlockWithoutSeal(chain, engine, chain.Genesis())
	stop := make(chan struct{}, 1)
	eventSub := engine.EventMux().Subscribe(istanbul.RequestEvent{})
	eventLoop := func() {
		select {
		case ev := <-eventSub.Chan():
			_, ok := ev.Data.(istanbul.RequestEvent)
			if !ok {
				t.Errorf("unexpected event comes: %v", reflect.TypeOf(ev.Data))
			}
			stop <- struct{}{}
		}
		eventSub.Unsubscribe()
	}
	go eventLoop()
	finalBlock, err := engine.SealWithReturn(chain, block, stop)
	if err != nil {
		t.Errorf("error mismatch: have %v, want nil", err)
	}
	if finalBlock != nil {
		t.Errorf("block mismatch: have %v, want nil", finalBlock)
	}
}

func TestSealCommittedOtherHash(t *testing.T) {
	chain, engine := newBlockChain(4)
	block := makeBlockWithoutSeal(chain, engine, chain.Genesis())
	otherBlock := makeBlockWithoutSeal(chain, engine, block)
	eventSub := engine.EventMux().Subscribe(istanbul.RequestEvent{})
	eventLoop := func() {
		select {
		case ev := <-eventSub.Chan():
			_, ok := ev.Data.(istanbul.RequestEvent)
			if !ok {
				t.Errorf("unexpected event comes: %v", reflect.TypeOf(ev.Data))
			}
			engine.Commit(otherBlock, [][]byte{})
		}
		eventSub.Unsubscribe()
	}
	go eventLoop()
	seal := func() {
		engine.SealWithReturn(chain, block, nil)
		t.Error("seal should not be completed")
	}
	go seal()

	const timeoutDura = 2 * time.Second
	timeout := time.NewTimer(timeoutDura)
	select {
	case <-timeout.C:
		// wait 2 seconds to ensure we cannot get any blocks from Istanbul
	}
}

func TestSealCommitted(t *testing.T) {
	chain, engine := newBlockChain(1)
	block := makeBlockWithoutSeal(chain, engine, chain.Genesis())
	expectedBlock, _ := engine.updateBlock(engine.chain.GetHeaderByHash(block.ParentHash()), block)

	finalBlock, err := engine.SealWithReturn(chain, block, nil)
	if err != nil {
		t.Errorf("error mismatch: have %v, want nil", err)
	}
	if finalBlock.Hash() != expectedBlock.Hash() {
		t.Errorf("hash mismatch: have %v, want %v", finalBlock.Hash(), expectedBlock.Hash())
	}
}

func TestVerifyHeader(t *testing.T) {
	chain, engine := newBlockChain(1)

	// errEmptyCommittedSeals case
	block := makeBlockWithoutSeal(chain, engine, chain.Genesis())
	block, _ = engine.updateBlock(chain.Genesis().Header, block)
	err := engine.VerifyHeader(chain, block.Header)
	if err != errEmptyCommittedSeals {
		t.Errorf("error mismatch: have %v, want %v", err, errEmptyCommittedSeals)
	}

	// short extra data
	header := block.Header
	header.ExtraData = []byte{}
	err = engine.VerifyHeader(chain, header)
	if err != errInvalidExtraDataFormat {
		t.Errorf("error mismatch: have %v, want %v", err, errInvalidExtraDataFormat)
	}
	// incorrect extra format
	header.ExtraData = []byte("0000000000000000000000000000000012300000000000000000000000000000000000000000000000000000000000000000")
	err = engine.VerifyHeader(chain, header)
	if err != errInvalidExtraDataFormat {
		t.Errorf("error mismatch: have %v, want %v", err, errInvalidExtraDataFormat)
	}

	// incorrect consensus type
	block = makeBlockWithoutSeal(chain, engine, chain.Genesis())
	header = block.Header
	header.Consensus = types.PowConsensus
	err = engine.VerifyHeader(chain, header)
	if err != errInvalidMixDigest {
		t.Errorf("error mismatch: have %v, want %v", err, errInvalidMixDigest)
	}

	// invalid difficulty
	block = makeBlockWithoutSeal(chain, engine, chain.Genesis())
	header = block.Header
	header.Difficulty = big.NewInt(2)
	err = engine.VerifyHeader(chain, header)
	if err != errInvalidDifficulty {
		t.Errorf("error mismatch: have %v, want %v", err, errInvalidDifficulty)
	}

	// invalid timestamp
	block = makeBlockWithoutSeal(chain, engine, chain.Genesis())
	header = block.Header
	header.CreateTimestamp = new(big.Int).Add(chain.Genesis().Time(), new(big.Int).SetUint64(engine.config.BlockPeriod-1))
	err = engine.VerifyHeader(chain, header)
	if err != errInvalidTimestamp {
		t.Errorf("error mismatch: have %v, want %v", err, errInvalidTimestamp)
	}

	// future block
	block = makeBlockWithoutSeal(chain, engine, chain.Genesis())
	header = block.Header
	header.CreateTimestamp = new(big.Int).Add(big.NewInt(now().Unix()), new(big.Int).SetUint64(10))
	err = engine.VerifyHeader(chain, header)
	if err != consensus.ErrBlockCreateTimeOld {
		t.Errorf("error mismatch: have %v, want %v", err, consensus.ErrBlockCreateTimeOld)
	}

	// invalid nonce
	block = makeBlockWithoutSeal(chain, engine, chain.Genesis())
	header = block.Header
	copy(header.Witness[:], hexutil.MustHexToBytes("0x111111111111"))
	header.Height = engine.config.Epoch
	err = engine.VerifyHeader(chain, header)
	if err != errInvalidNonce {
		t.Errorf("error mismatch: have %v, want %v", err, errInvalidNonce)
	}
}

func TestVerifySeal(t *testing.T) {
	chain, engine := newBlockChain(1)
	genesis := chain.Genesis()
	// cannot verify genesis
	err := engine.VerifySeal(chain, genesis.Header)
	if err != errUnknownBlock {
		t.Errorf("error mismatch: have %v, want %v", err, errUnknownBlock)
	}

	block := makeBlock(chain, engine, genesis)
	// change block content
	header := block.Header
	header.Height = 4
	block1 := block.WithSeal(header)
	err = engine.VerifySeal(chain, block1.Header)
	if err != errUnauthorized {
		t.Errorf("error mismatch: have %v, want %v", err, errUnauthorized)
	}

	// unauthorized users but still can get correct signer address
	engine.privateKey, _ = crypto.GenerateKey()
	err = engine.VerifySeal(chain, block.Header)
	if err != nil {
		t.Errorf("error mismatch: have %v, want nil", err)
	}
}

func TestVerifyHeaders(t *testing.T) {
	chain, engine := newBlockChain(1)
	genesis := chain.Genesis()

	// success case
	headers := []*types.BlockHeader{}
	blocks := []*types.Block{}
	size := 100

	for i := 0; i < size; i++ {
		var b *types.Block
		if i == 0 {
			b = makeBlockWithoutSeal(chain, engine, genesis)
			b, _ = engine.updateBlock(genesis.Header, b)
		} else {
			b = makeBlockWithoutSeal(chain, engine, blocks[i-1])
			b, _ = engine.updateBlock(blocks[i-1].Header, b)
		}
		blocks = append(blocks, b)
		headers = append(headers, blocks[i].Header)
	}
	now = func() time.Time {
		return time.Unix(headers[size-1].CreateTimestamp.Int64(), 0)
	}
	_, results := engine.VerifyHeaders(chain, headers, nil)
	const timeoutDura = 2 * time.Second
	timeout := time.NewTimer(timeoutDura)
	index := 0
OUT1:
	for {
		select {
		case err := <-results:
			if err != nil {
				if err != errEmptyCommittedSeals && err != errInvalidCommittedSeals {
					t.Errorf("error mismatch: have %v, want errEmptyCommittedSeals|errInvalidCommittedSeals", err)
					break OUT1
				}
			}
			index++
			if index == size {
				break OUT1
			}
		case <-timeout.C:
			break OUT1
		}
	}
	// abort cases
	abort, results := engine.VerifyHeaders(chain, headers, nil)
	timeout = time.NewTimer(timeoutDura)
	index = 0
OUT2:
	for {
		select {
		case err := <-results:
			if err != nil {
				if err != errEmptyCommittedSeals && err != errInvalidCommittedSeals {
					t.Errorf("error mismatch: have %v, want errEmptyCommittedSeals|errInvalidCommittedSeals", err)
					break OUT2
				}
			}
			index++
			if index == 5 {
				abort <- struct{}{}
			}
			if index >= size {
				t.Errorf("verifyheaders should be aborted")
				break OUT2
			}
		case <-timeout.C:
			break OUT2
		}
	}
	// error header cases
	headers[2].Height = 100
	abort, results = engine.VerifyHeaders(chain, headers, nil)
	timeout = time.NewTimer(timeoutDura)
	index = 0
	errors := 0
	expectedErrors := 2
OUT3:
	for {
		select {
		case err := <-results:
			if err != nil {
				if err != errEmptyCommittedSeals && err != errInvalidCommittedSeals {
					errors++
				}
			}
			index++
			if index == size {
				if errors != expectedErrors {
					t.Errorf("error mismatch: have %v, want %v", err, expectedErrors)
				}
				break OUT3
			}
		case <-timeout.C:
			break OUT3
		}
	}
}

func TestPrepareExtra(t *testing.T) {
	validators := make([]common.Address, 4)
	validators[0] = common.BytesToAddress(hexutil.MustHexToBytes("0x44add0ec310f115a0e603b2d7db9f067778eaf8a"))
	validators[1] = common.BytesToAddress(hexutil.MustHexToBytes("0x294fc7e8f22b3bcdcf955dd7ff3ba2ed833f8212"))
	validators[2] = common.BytesToAddress(hexutil.MustHexToBytes("0x6beaaed781d2d2ab6350f5c4566a2c6eaac407a6"))
	validators[3] = common.BytesToAddress(hexutil.MustHexToBytes("0x8be76812f765c24641ec63dc2852b378aba2b440"))

	vanity := make([]byte, types.IstanbulExtraVanity)
	expectedResult := append(vanity, hexutil.MustHexToBytes("0xf858f8549444add0ec310f115a0e603b2d7db9f067778eaf8a94294fc7e8f22b3bcdcf955dd7ff3ba2ed833f8212946beaaed781d2d2ab6350f5c4566a2c6eaac407a6948be76812f765c24641ec63dc2852b378aba2b44080c0")...)

	h := &types.BlockHeader{
		ExtraData: vanity,
	}

	payload, err := prepareExtra(h, validators)
	if err != nil {
		t.Errorf("error mismatch: have %v, want: nil", err)
	}
	if !reflect.DeepEqual(payload, expectedResult) {
		t.Errorf("payload mismatch: have %v, want %v", payload, expectedResult)
	}

	// append useless information to extra-data
	h.ExtraData = append(vanity, make([]byte, 15)...)

	payload, err = prepareExtra(h, validators)
	if !reflect.DeepEqual(payload, expectedResult) {
		t.Errorf("payload mismatch: have %v, want %v", payload, expectedResult)
	}
}

func TestWriteSeal(t *testing.T) {
	vanity := bytes.Repeat([]byte{0x00}, types.IstanbulExtraVanity)
	istRawData := hexutil.MustHexToBytes("0xf858f8549444add0ec310f115a0e603b2d7db9f067778eaf8a94294fc7e8f22b3bcdcf955dd7ff3ba2ed833f8212946beaaed781d2d2ab6350f5c4566a2c6eaac407a6948be76812f765c24641ec63dc2852b378aba2b44080c0")
	expectedSeal := append([]byte{1, 2, 3}, bytes.Repeat([]byte{0x00}, types.IstanbulExtraSeal-3)...)
	expectedIstExtra := &types.IstanbulExtra{
		Validators: []common.Address{
			common.BytesToAddress(hexutil.MustHexToBytes("0x44add0ec310f115a0e603b2d7db9f067778eaf8a")),
			common.BytesToAddress(hexutil.MustHexToBytes("0x294fc7e8f22b3bcdcf955dd7ff3ba2ed833f8212")),
			common.BytesToAddress(hexutil.MustHexToBytes("0x6beaaed781d2d2ab6350f5c4566a2c6eaac407a6")),
			common.BytesToAddress(hexutil.MustHexToBytes("0x8be76812f765c24641ec63dc2852b378aba2b440")),
		},
		Seal:          expectedSeal,
		CommittedSeal: [][]byte{},
	}
	var expectedErr error

	h := &types.BlockHeader{
		ExtraData: append(vanity, istRawData...),
	}

	// normal case
	err := writeSeal(h, expectedSeal)
	if err != expectedErr {
		t.Errorf("error mismatch: have %v, want %v", err, expectedErr)
	}

	// verify istanbul extra-data
	istExtra, err := types.ExtractIstanbulExtra(h)
	if err != nil {
		t.Errorf("error mismatch: have %v, want nil", err)
	}
	if !reflect.DeepEqual(istExtra, expectedIstExtra) {
		t.Errorf("extra data mismatch: have %v, want %v", istExtra, expectedIstExtra)
	}

	// invalid seal
	unexpectedSeal := append(expectedSeal, make([]byte, 1)...)
	err = writeSeal(h, unexpectedSeal)
	if err != errInvalidSignature {
		t.Errorf("error mismatch: have %v, want %v", err, errInvalidSignature)
	}
}

func TestWriteCommittedSeals(t *testing.T) {
	vanity := bytes.Repeat([]byte{0x00}, types.IstanbulExtraVanity)
	istRawData := hexutil.MustHexToBytes("0xf858f8549444add0ec310f115a0e603b2d7db9f067778eaf8a94294fc7e8f22b3bcdcf955dd7ff3ba2ed833f8212946beaaed781d2d2ab6350f5c4566a2c6eaac407a6948be76812f765c24641ec63dc2852b378aba2b44080c0")
	expectedCommittedSeal := append([]byte{1, 2, 3}, bytes.Repeat([]byte{0x00}, types.IstanbulExtraSeal-3)...)
	expectedIstExtra := &types.IstanbulExtra{
		Validators: []common.Address{
			common.BytesToAddress(hexutil.MustHexToBytes("0x44add0ec310f115a0e603b2d7db9f067778eaf8a")),
			common.BytesToAddress(hexutil.MustHexToBytes("0x294fc7e8f22b3bcdcf955dd7ff3ba2ed833f8212")),
			common.BytesToAddress(hexutil.MustHexToBytes("0x6beaaed781d2d2ab6350f5c4566a2c6eaac407a6")),
			common.BytesToAddress(hexutil.MustHexToBytes("0x8be76812f765c24641ec63dc2852b378aba2b440")),
		},
		Seal:          []byte{},
		CommittedSeal: [][]byte{expectedCommittedSeal},
	}
	var expectedErr error

	h := &types.BlockHeader{
		ExtraData: append(vanity, istRawData...),
	}

	// normal case
	err := writeCommittedSeals(h, [][]byte{expectedCommittedSeal})
	if err != expectedErr {
		t.Errorf("error mismatch: have %v, want %v", err, expectedErr)
	}

	// verify istanbul extra-data
	istExtra, err := types.ExtractIstanbulExtra(h)
	if err != nil {
		t.Errorf("error mismatch: have %v, want nil", err)
	}
	if !reflect.DeepEqual(istExtra, expectedIstExtra) {
		t.Errorf("extra data mismatch: have %v, want %v", istExtra, expectedIstExtra)
	}

	// invalid seal
	unexpectedCommittedSeal := append(expectedCommittedSeal, make([]byte, 1)...)
	err = writeCommittedSeals(h, [][]byte{unexpectedCommittedSeal})
	if err != errInvalidCommittedSeals {
		t.Errorf("error mismatch: have %v, want %v", err, errInvalidCommittedSeals)
	}
}
