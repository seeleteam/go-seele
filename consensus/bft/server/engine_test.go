/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package server

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"reflect"

	"testing"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/consensus/bft"
	bftCore "github.com/seeleteam/go-seele/consensus/bft/core"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/txs"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database/leveldb"
)

var genesisAccount = crypto.MustGenerateShardAddress(1)

func newTestGenesis(n int) (*core.Genesis, []*ecdsa.PrivateKey) {
	accounts := map[common.Address]*big.Int{
		*genesisAccount: new(big.Int).Mul(big.NewInt(4), common.SeeleToFan),
	}

	// Setup verifiers
	var nodeKeys = make([]*ecdsa.PrivateKey, n)
	var addrs = make([]common.Address, n)
	for i := 0; i < n; i++ {
		nodeKeys[i], _ = crypto.GenerateKey()
		addrs[i] = crypto.PubkeyToAddress(nodeKeys[i].PublicKey)
	}

	genesis := core.GetGenesis(core.NewGenesisInfo(accounts, 1, 1, big.NewInt(0), types.BftConsensus, addrs))
	fmt.Println("genesis", genesis, "nodeKeys", nodeKeys)
	return genesis, nodeKeys
}

// in this test, we can set n to 1, and it means we can process bft and commit a
// block by one node. Otherwise, if n is larger than 1, we have to generate
// other fake events to process bft.
func newBlockChain(n int) (*core.Blockchain, *server) {
	// db
	db, _ := leveldb.NewTestDatabase()
	bcStore := store.NewCachedStore(store.NewBlockchainDatabase(db))

	// genesis
	genesis, nodeKeys := newTestGenesis(n)
	if err := genesis.InitializeAndValidate(bcStore, db); err != nil {
		panic(err)
	} else {
		fmt.Println("sucessfully InitializedAndValidate")
	}

	config := bft.DefaultConfig
	b, _ := NewServer(config, nodeKeys[0], db).(*server)

	bc, err := core.NewBlockchain(bcStore, db, "", b, nil, 0)
	if err != nil {
		fmt.Println("NewBlockchain err", err)
		panic(err)
	}

	b.Start(bc, bc.CurrentBlock, func(hash common.Hash) bool {
		return false
	})

	snap, err := b.snapshot(bc, 0, common.Hash{}, nil)
	if err != nil {
		fmt.Println("snapshot err", err)
		panic(err)
	}
	if snap == nil {
		panic("failed to get snapshot")
	} else {
		fmt.Println("snap", snap)
	}
	proposer := snap.VerSet.GetProposer()
	if proposer != nil {
		fmt.Println("proposer", proposer)
	} else {
		fmt.Println("proposer is nil")
		return nil, nil
	}
	fmt.Println("snap.VerSet.GetProposer().Address()", snap.VerSet.GetProposer().Address())
	proposerAddr := snap.VerSet.GetProposer().Address()

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

func makeHeader(parent *types.Block, config *bft.BFTConfig) *types.BlockHeader {
	header := &types.BlockHeader{
		PreviousBlockHash: parent.Header.Hash(),
		Height:            parent.Height() + 1,
		CreateTimestamp:   new(big.Int).Add(parent.Header.CreateTimestamp, new(big.Int).SetUint64(config.BlockPeriod)),
		Difficulty:        defaultDifficulty,
		ExtraData:         parent.Header.ExtraData,
		StateHash:         parent.Header.StateHash,
		Witness:           make([]byte, bft.WitnessSize),
	}
	return header
}

// func newBlock(chain *core.Blockchain, engine *server, parent *types.Block) *types.Block {
// 	block := newBlockWithoutSeal(chain, engine, parent)
// 	fmt.Printf("after newBlockWithoutSeal block %+v", block)
// 	stop := make(chan struct{})
// 	block, err := engine.SealResult(chain, block, stop) // here stop chan is nil, will panic with test timed out after 30s
// 	fmt.Printf("newBlock block %+v err %+v\n", block, err)
// 	if err != nil {
// 		fmt.Println("newBlock err", err)
// 		panic(err)
// 	}

// 	return block
// }

// newBlock make a block with seals
// first create block without seals
// then seal the result inside the block
func newBlock(chain *core.Blockchain, engine *server, parent *types.Block) *types.Block {
	block := newBlockWithoutSeal(chain, engine, parent)
	fmt.Printf("[4-1]newBlock newBlockWithoutSeal %+v", block)
	// this does not return result !!!
	block, _ = engine.SealResult(chain, block, nil)
	fmt.Printf("[4-2]newBlock SealResult %+v", block)

	return block
}

// newBlockWithoutSeal make block without seals.
// by the time of proposer seal calculation,
// the committed seals are still unknown,
// so we calculate the seal with those unknowns empty.
func newBlockWithoutSeal(chain *core.Blockchain, engine *server, parent *types.Block) *types.Block {
	header := makeHeader(parent, engine.config)
	// prepare block
	engine.Prepare(chain, header)
	// reward tx
	reward := consensus.GetReward(header.Height)
	rewardTx, err := txs.NewRewardTx(header.Creator, reward, header.CreateTimestamp.Uint64())
	if err != nil {
		panic(err)
	}

	state, err := chain.GetState(parent.Header.StateHash)
	if err != nil {
		panic(err)
	}

	rewardTxReceipt, err := txs.ApplyRewardTx(rewardTx, state)

	// new statehash with rewardtx
	header.StateHash, err = state.Hash()
	if err != nil {
		panic(err)
	}

	// make a new block
	block := types.NewBlock(header, []*types.Transaction{rewardTx}, []*types.Receipt{rewardTxReceipt}, nil)
	return block
}

// ok  	github.com/seeleteam/go-seele/consensus/bft/server	0.085s
// Success: Tests passed.
func TestSealStopChannel(t *testing.T) {
	// var timeStart uint64
	// timeStart = time.Now().Unix()
	chain, engine := newBlockChain(4)
	block := newBlockWithoutSeal(chain, engine, chain.Genesis())
	stop := make(chan struct{}, 1)
	eventSub := engine.EventMux().Subscribe(bft.RequestEvent{})
	// fmt.Println("time duration", time.Now().Unix()-timeStart)
	eventLoop := func() {
		select {
		case ev := <-eventSub.Chan():
			_, ok := ev.Data.(bft.RequestEvent)
			if !ok {
				t.Errorf("unexpected event comes: %v", reflect.TypeOf(ev.Data))
			}
			stop <- struct{}{}
		}
		eventSub.Unsubscribe()
	}
	go eventLoop()
	finalBlock, err := engine.SealResult(chain, block, stop)
	if err != nil {
		t.Errorf("error mismatch: have %v, want nil", err)
	}
	if finalBlock != nil {
		t.Errorf("block mismatch: have %v, want nil", finalBlock)
	}
}

//--- FAIL: TestSealCommittedOtherHash (0.04s)
// panic: failed to validate block ===>  ===> failed to verify header by consensus engine ===> committed seals are invalid
func TestSealCommittedOtherHash(t *testing.T) {
	chain, engine := newBlockChain(1)
	block := newBlockWithoutSeal(chain, engine, chain.Genesis())
	b, err := engine.updateBlock(block.Header, block)
	if err != nil {
		fmt.Println("updateBlock err", err)
		panic(err)
	}

	prepareCommittedSeal := bftCore.PrepareCommittedSeal(b.HeaderHash)
	CommittedSeal, err := engine.Sign(prepareCommittedSeal)
	if err != nil {
		panic(err)
	}
	err = writeCommittedSeals(b.Header, [][]byte{CommittedSeal})
	if err != nil {
		panic(err)
	}
	// update block's header
	b = b.WithSeal(b.Header)

	// FIXME : WriteBlock return err!
	err = chain.WriteBlock(b)
	if err != nil {
		fmt.Println("WriteBlock err", err)
		panic(err)
	}
	otherBlock := newBlockWithoutSeal(chain, engine, b)
	eventSub := engine.EventMux().Subscribe(bft.RequestEvent{})
	eventLoop := func() {
		select {
		case ev := <-eventSub.Chan():
			_, ok := ev.Data.(bft.RequestEvent)
			if !ok {
				t.Errorf("unexpected event comes: %v", reflect.TypeOf(ev.Data))
			}
			engine.Commit(otherBlock, [][]byte{})
		}
		eventSub.Unsubscribe()
	}
	go eventLoop()
	seal := func() {
		engine.Seal(chain, otherBlock, nil, nil)
		t.Error("seal should not be completed")
	}
	go seal()

	const timeoutDuration = 2 * time.Second
	timeout := time.NewTimer(timeoutDuration)
	select {
	case <-timeout.C:
		// wait 2 seconds to ensure we cannot get any blocks from bft
	}
}

/*
panic: test timed out after 30s
*/
func TestSealCommitted(t *testing.T) {
	chain, engine := newBlockChain(1)
	block := newBlockWithoutSeal(chain, engine, chain.Genesis())
	expectedBlock, _ := engine.updateBlock(engine.chain.GetHeaderByHash(block.ParentHash()), block)

	finalBlock, err := engine.SealResult(chain, block, nil)
	if err != nil {
		t.Errorf("error mismatch: have [%v], want [nil]", err)
	}
	if finalBlock.Hash() != expectedBlock.Hash() {
		t.Errorf("hash mismatch: have %v, want %v", finalBlock.Hash(), expectedBlock.Hash())
	}
}

/*
ok  	github.com/seeleteam/go-seele/consensus/bft/server	0.093s
Success: Tests passed.
*/
func TestVerifyHeader(t *testing.T) {
	chain, engine := newBlockChain(1)

	// errEmptyCommittedSeals case
	block := newBlockWithoutSeal(chain, engine, chain.Genesis())
	block, _ = engine.updateBlock(chain.Genesis().Header, block)
	err := engine.VerifyHeader(chain, block.Header)
	if err != errEmptyCommittedSeals {
		t.Errorf("error mismatch: have %v, want %v", err, errEmptyCommittedSeals)
	}

	// short extra data
	header := block.Header
	header.ExtraData = []byte{}
	err = engine.VerifyHeader(chain, header)
	fmt.Printf("[1]verifyHeader (before extraData) with err %+v\n", err)
	if err != errExtraDataFormatInvalid {
		t.Errorf("error mismatch: have %v, want %v", err, errExtraDataFormatInvalid)
	}
	// incorrect extra format
	header.ExtraData = []byte("0000000000000000000000000000000012300000000000000000000000000000000000000000000000000000000000000000")
	err = engine.VerifyHeader(chain, header)
	fmt.Printf("[2]verifyHeader (after extraData) with err %+v\n", err)

	if err != errExtraDataFormatInvalid {
		t.Errorf("error mismatch: have %v, want %v", err, errExtraDataFormatInvalid)
	}

	// incorrect consensus type
	block = newBlockWithoutSeal(chain, engine, chain.Genesis())
	header = block.Header
	header.Consensus = types.PowConsensus // assign a incorrect consensus
	// header.Consensus = types.BftConsensus
	err = engine.VerifyHeader(chain, header)
	fmt.Printf("[3]verifyHeader (after consensus) with err %+v\n", err)

	if err != errBFTConsensus {
		t.Errorf("error mismatch: have %v, want %v", err, errBFTConsensus)
	}

	// invalid difficulty
	block = newBlockWithoutSeal(chain, engine, chain.Genesis())
	header = block.Header
	header.Difficulty = big.NewInt(2)
	err = engine.VerifyHeader(chain, header)

	fmt.Printf("[4]verifyHeader (after difficulty) with err %+v\n", err)

	if err != errDifficultyInvalid {
		t.Errorf("error mismatch: have %v, want %v", err, errDifficultyInvalid)
	}

	// invalid timestamp
	block = newBlockWithoutSeal(chain, engine, chain.Genesis())
	header = block.Header
	header.CreateTimestamp = new(big.Int).Add(chain.Genesis().Time(), new(big.Int).SetUint64(engine.config.BlockPeriod-1))
	err = engine.VerifyHeader(chain, header)
	fmt.Printf("[5]verifyHeader (after invalid timeStamp) with err %+v\n", err)

	if err != errTimestampInvalid {
		t.Errorf("error mismatch: have %v, want %v", err, errTimestampInvalid)
	}

	// future block
	block = newBlockWithoutSeal(chain, engine, chain.Genesis())
	header = block.Header
	header.CreateTimestamp = new(big.Int).Add(big.NewInt(now().Unix()), new(big.Int).SetUint64(10))
	err = engine.VerifyHeader(chain, header)
	fmt.Printf("[6]verifyHeader (after future timeStamp) with err %+v\n", err)

	if err != consensus.ErrBlockCreateTimeOld {
		t.Errorf("error mismatch: have %v, want %v", err, consensus.ErrBlockCreateTimeOld)
	}

	//invalid nonce
	block = newBlockWithoutSeal(chain, engine, chain.Genesis())
	header = block.Header
	copy(header.Witness[:], hexutil.MustHexToBytes("0x111111111111"))
	header.Height = engine.config.Epoch
	err = engine.VerifyHeader(chain, header)
	fmt.Printf("[7]verifyHeader (after invalid nonce) with err %+v\n", err)

	if err != errNonceInvalid {
		t.Errorf("error mismatch: have %v, want %v", err, errNonceInvalid)
	}
}

/*
panic: test timed out after 30s
github.com/seeleteam/go-seele/consensus/bft/server.(*server).SealResult(0xc000118120, 0x463ac00, 0xc000114fa0, 0xc0000948a0, 0x0, 0x0, 0x0, 0x0)
	/Users/seele/go/src/github.com/seeleteam/go-seele/consensus/bft/server/engine.go:202 +0x58c
github.com/seeleteam/go-seele/consensus/bft/server.newBlock(0xc0001f43c0, 0xc00011a120, 0xc0000ce600, 0xc000154701)
	/Users/seele/go/src/github.com/seeleteam/go-seele/consensus/bft/server/engine_test.go:126 +0x70
github.com/seeleteam/go-seele/consensus/bft/server.TestVerifySeal(0xc0001b8100)
	/Users/seele/go/src/github.com/seeleteam/go-seele/consensus/bft/server/engine_test.go:358 +0x152
*/
func TestVerifySeal(t *testing.T) {
	chain, engine := newBlockChain(1) //generate 1 node in verifier set
	fmt.Printf("[1]new BlockChain chain %+v\n", chain)
	fmt.Printf("[1]new BlockChain engine %+v\n", engine)
	genesis := chain.Genesis()
	fmt.Printf("[2]genesis %+v\n", genesis)
	// cannot verify genesis
	err := engine.VerifySeal(chain, genesis.Header) // since the height = 0 for genesis, this will give back that height = 0 err!
	fmt.Println("TestVerifySeal should return [unknown block], actutally return", err)
	if err != errBlockUnknown {
		t.Errorf("error mismatch: have %v, want %v", err, errBlockUnknown)
	}

	fmt.Println("[3]begin to make block")
	// here newBlock there is no commit at all.
	block := newBlock(chain, engine, genesis)
	fmt.Printf("[4-done]newBlock %+v\n", block)
	// change block content
	header := block.Header.Clone()
	fmt.Printf("header %+v\n", header)
	header.Height = 4
	fmt.Printf("header %+v\n", header)

	block1 := block.WithSeal(header)
	err2 := engine.VerifySeal(chain, block1.Header)
	fmt.Println("TestVerifySeal should return [unauthorized], actutally return", err2)
	if err2 != errUnauthorized {
		t.Errorf("error mismatch: have %v, want %v", err2, errUnauthorized)
	}

	// unauthorized users but still can get correct signer address
	engine.privateKey, _ = crypto.GenerateKey()
	err3 := engine.VerifySeal(chain, block.Header)
	if err != nil {
		t.Errorf("error mismatch: have %v, want nil", err3)
	}
}

/*
ok  	github.com/seeleteam/go-seele/consensus/bft/server	0.033s
Success: Tests passed.
*/
func TestPrepareExtra(t *testing.T) {
	// verifier list
	verifiers := make([]common.Address, 4)
	verifiers[0] = common.BytesToAddress(hexutil.MustHexToBytes("0x44add0ec310f115a0e603b2d7db9f067778eaf8a"))
	verifiers[1] = common.BytesToAddress(hexutil.MustHexToBytes("0x294fc7e8f22b3bcdcf955dd7ff3ba2ed833f8212"))
	verifiers[2] = common.BytesToAddress(hexutil.MustHexToBytes("0x6beaaed781d2d2ab6350f5c4566a2c6eaac407a6"))
	verifiers[3] = common.BytesToAddress(hexutil.MustHexToBytes("0x8be76812f765c24641ec63dc2852b378aba2b440"))

	vanity := make([]byte, types.BftExtraVanity)
	expectedResult := append(vanity, hexutil.MustHexToBytes("0xf858f8549444add0ec310f115a0e603b2d7db9f067778eaf8a94294fc7e8f22b3bcdcf955dd7ff3ba2ed833f8212946beaaed781d2d2ab6350f5c4566a2c6eaac407a6948be76812f765c24641ec63dc2852b378aba2b44080c0")...)

	h := &types.BlockHeader{
		ExtraData: vanity,
	}

	payload, err := prepareExtra(h, verifiers)
	if err != nil {
		t.Errorf("error mismatch: have %v, want: nil", err)
	}
	if !reflect.DeepEqual(payload, expectedResult) {
		t.Errorf("payload mismatch: have %v, want %v", payload, expectedResult)
	}

	// append useless information to extra-data
	h.ExtraData = append(vanity, make([]byte, 15)...)

	payload, err = prepareExtra(h, verifiers)
	if !reflect.DeepEqual(payload, expectedResult) {
		t.Errorf("payload mismatch: have %v, want %v", payload, expectedResult)
	}
}

/*
ok  	github.com/seeleteam/go-seele/consensus/bft/server	0.035s
Success: Tests passed.
*/
func TestWriteSeal(t *testing.T) {
	vanity := bytes.Repeat([]byte{0x00}, types.BftExtraVanity)
	istRawData := hexutil.MustHexToBytes("0xf858f8549444add0ec310f115a0e603b2d7db9f067778eaf8a94294fc7e8f22b3bcdcf955dd7ff3ba2ed833f8212946beaaed781d2d2ab6350f5c4566a2c6eaac407a6948be76812f765c24641ec63dc2852b378aba2b44080c0")
	expectedSeal := append([]byte{1, 2, 3}, bytes.Repeat([]byte{0x00}, types.BftExtraSeal-3)...)
	expectedIstExtra := &types.BftExtra{
		Verifiers: []common.Address{
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

	// verify bft extra-data
	istExtra, err := types.ExtractBftExtra(h)
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

/*
ok  	github.com/seeleteam/go-seele/consensus/bft/server	0.045s
Success: Tests passed.
*/
func TestWriteCommittedSeals(t *testing.T) {
	vanity := bytes.Repeat([]byte{0x00}, types.BftExtraVanity)
	istRawData := hexutil.MustHexToBytes("0xf858f8549444add0ec310f115a0e603b2d7db9f067778eaf8a94294fc7e8f22b3bcdcf955dd7ff3ba2ed833f8212946beaaed781d2d2ab6350f5c4566a2c6eaac407a6948be76812f765c24641ec63dc2852b378aba2b44080c0")
	expectedCommittedSeal := append([]byte{1, 2, 3}, bytes.Repeat([]byte{0x00}, types.BftExtraSeal-3)...)
	expectedIstExtra := &types.BftExtra{
		Verifiers: []common.Address{
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

	// verify bft extra-data
	istExtra, err := types.ExtractBftExtra(h)
	if err != nil {
		t.Errorf("error mismatch: have %v, want nil", err)
	}
	if !reflect.DeepEqual(istExtra, expectedIstExtra) {
		t.Errorf("extra data mismatch: have %v, want %v", istExtra, expectedIstExtra)
	}

	// invalid seal
	unexpectedCommittedSeal := append(expectedCommittedSeal, make([]byte, 1)...)
	err = writeCommittedSeals(h, [][]byte{unexpectedCommittedSeal})
	if err != errCommittedSealsInvalid {
		t.Errorf("error mismatch: have %v, want %v", err, errCommittedSealsInvalid)
	}
}

/*
ok  	github.com/seeleteam/go-seele/consensus/bft/server	0.064s
Success: Tests passed.
*/
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
