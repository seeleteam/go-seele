/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package system

import (
	"encoding/json"
	"fmt"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core/state"
)

const (
	// CmdVerifyTx is the command byte to verify btc tx
	CmdVerifyTx = iota
	// CmdRelayTx is the command byte to relay btc tx
	CmdRelayTx
	// CmdStoreBlockHeader is the command byte to store btc block header
	CmdStoreBlockHeader
	// CmdGetBlockHeader is the command byte to get btc block header
	CmdGetBlockHeader
)

const btcConfirm = 6

var (
	brCommands = map[byte]*cmdInfo{
		CmdVerifyTx:         &cmdInfo{1000, verifyTx},
		CmdRelayTx:          &cmdInfo{3000, relayTx},
		CmdStoreBlockHeader: &cmdInfo{10000, storeBlockHeader},
		CmdGetBlockHeader:   &cmdInfo{200, getBlockHeader},
	}

	// function result
	failure = []byte{0}
	success = []byte{1}

	// temp storage
	preBlockHeight uint64
	keyBlocksHash  = common.BytesToHash([]byte("BTC-Blocks-Hash"))
	// @todo will change after the relay chain is successfully established.
	fee uint64 = 1
)

// BTCBlock btc block structure
type BTCBlock struct {
	BlockHeaderHex   string
	Height           uint64
	PreviousBlockHex string
	TxHexs           []string
	Relayer          common.Address
}

func (b *BTCBlock) String() string {
	return fmt.Sprintf("Block[BlockHeaderHex=%v, Height=%v, PreviousBlockHex=%v, TxHexs=%v]", b.BlockHeaderHex, b.Height, b.PreviousBlockHex, b.TxHexs)
}

// RelayRequest is a request structure using btc-relay
type RelayRequest struct {
	BTCBlock
	TxHex string
	// This is used to relay the verification of successful tx to the contract address
	RelayAddress common.Address
}

func (r *RelayRequest) String() string {
	return fmt.Sprintf("RelayRequest[BTCBlock=%v, TxHex=%v, RelayAddress=%v]", r.BTCBlock.String(), r.TxHex, r.RelayAddress.Hex())
}

// verify that tx exists based on request, the amount will be transferred
// to BTCRelayContractAddress if it doesn't match the block. When it
// matches the block, the amount will be transferred to the relayer. If an
// error occurs, the amount will be reverted to user(from).
func verifyTx(request []byte, ctx *Context) ([]byte, error) {
	var relayRequest RelayRequest
	if err := json.Unmarshal(request, &relayRequest); err != nil {
		return failure, fmt.Errorf("Invalid request parameter, %s", err)
	}

	amount := ctx.tx.Data.Amount
	if amount.Uint64() < fee {
		return failure, fmt.Errorf("Verify tx fee is not enough, expected[%d], actual[%d]", fee, amount)
	}

	// Match block
	blocks := getBTCBlocks(ctx.statedb)
	btcBlock, ok := blocks[relayRequest.BlockHeaderHex]
	if !ok {
		return failure, nil
	}

	if preBlockHeight < btcConfirm || btcBlock.Height < preBlockHeight-btcConfirm {
		return failure, fmt.Errorf("Confirmation need more than 6, latestHeight[%d], queryHeight[%d]", preBlockHeight, btcBlock.Height)
	}

	// Tranfer amount to Relayer
	ctx.statedb.AddBalance(btcBlock.Relayer, amount)
	ctx.statedb.SubBalance(BTCRelayContractAddress, amount)

	for _, txHex := range btcBlock.TxHexs {
		if relayRequest.TxHex == txHex {
			return success, nil
		}
	}

	return failure, nil
}

// optionally relay the btc transaction to any Seele contract
func relayTx(request []byte, ctx *Context) ([]byte, error) {
	ok, err := verifyTx(request, ctx)
	if err != nil {
		return failure, err
	}

	if len(ok) == len(success) && success[0] == ok[0] {
		//@todo transfer tx to relay address
		return success, nil
	}

	return failure, nil
}

// storage of Bitcoin block headers
func storeBlockHeader(request []byte, ctx *Context) ([]byte, error) {
	if !isRelayer(ctx.tx.Data.From) {
		return failure, fmt.Errorf("Invaild block relayer[%s]", ctx.tx.Data.From.Hex())
	}

	var relayRequest RelayRequest
	if err := json.Unmarshal(request, &relayRequest); err != nil {
		return failure, fmt.Errorf("Invalid request parameter, %s", err)
	}

	blocks := getBTCBlocks(ctx.statedb)
	if _, ok := blocks[relayRequest.BlockHeaderHex]; ok {
		return failure, fmt.Errorf("Block header already exists")
	}

	relayRequest.BTCBlock.Relayer = ctx.tx.Data.From
	blocks[relayRequest.BlockHeaderHex] = relayRequest.BTCBlock
	preBlockHeight = relayRequest.Height

	bytes, err := json.Marshal(blocks)
	if err != nil {
		return failure, fmt.Errorf("Failed to marshal blocks, %s", err)
	}

	ctx.statedb.SetData(BTCRelayContractAddress, keyBlocksHash, bytes)
	return success, nil
}

// check if there is a stored bitcoin block header in the contract
func getBlockHeader(request []byte, ctx *Context) ([]byte, error) {
	amount := ctx.tx.Data.Amount
	if amount.Uint64() < fee {
		return failure, fmt.Errorf("getBlockHeader fee is not enough, expected[%d], actual[%d]", fee, amount)
	}

	blocks, blockHeaderHex := getBTCBlocks(ctx.statedb), hexutil.BytesToHex(request)
	btcBlock, ok := blocks[blockHeaderHex]
	if !ok {
		return failure, nil
	}

	// Tranfer amount to Relayer
	ctx.statedb.AddBalance(btcBlock.Relayer, amount)
	ctx.statedb.SubBalance(BTCRelayContractAddress, amount)

	return success, nil
}

// @todo
func isRelayer(addr common.Address) bool {
	return true
}

func getBTCBlocks(statedb *state.Statedb) map[string]BTCBlock {
	var blocks map[string]BTCBlock
	data := statedb.GetData(BTCRelayContractAddress, keyBlocksHash)
	if err := json.Unmarshal(data, &blocks); err != nil {
		return make(map[string]BTCBlock)
	}

	return blocks
}
