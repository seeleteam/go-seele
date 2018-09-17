/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package system

import (
	"math/big"

	"github.com/pkg/errors"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/crypto"
)

const (
	// CmdDeposit deposit seele and register as masternode
	CmdDeposit byte = iota
	// CmdQueryMasternode query masternode
	CmdQueryMasternode
	// CmdRecall recallCmd seele and unregister as masternode
	CmdRecall
	// CmdQuit quitCmd masternode
	CmdQuit

	gasCmdDeposit         = uint64(50000) // gas used to deposit
	gasCmdQueryMasterNode = uint64(5000)  // gas used to query masternode
	gasCmdRecall          = uint64(50000) // gas used to recallCmd
	gasCmdQuit            = uint64(50000) // gas used to quitCmd
)

var (
	ByteTrue  = []byte{1}
	ByteFalse = []byte{0}

	depositLimit        = big.NewInt(0).Mul(common.SeeleToFan, big.NewInt(20000))
	recallDistanceLimit = uint64(8640) // generate blocks in about one day
)

var (
	ErrDepositNotRight   = errors.New("deposit amount is not right")
	ErrAlreadyExist      = errors.New("this address is already masternode")
	ErrNotExist          = errors.New("this address is not masternode")
	ErrNotQuit           = errors.New("address doesn't quit")
	ErrNotEnoughDistance = errors.New("not enough distance")

	masternodeCommands = map[byte]*cmdInfo{
		CmdDeposit:         &cmdInfo{gasCmdDeposit, deposit},
		CmdQueryMasternode: &cmdInfo{gasCmdQueryMasterNode, queryMasternodeCmd},
		CmdRecall:          {gasCmdRecall, recallCmd},
		CmdQuit:            {gasCmdQuit, quitCmd},
	}
)

type masternodeInfo struct {
	IsQuit    bool
	QuitBlock uint64
}

func deposit(input []byte, context *Context) ([]byte, error) {
	if context.tx.Data.Amount.Cmp(depositLimit) != 0 {
		return nil, ErrDepositNotRight
	}

	sender := context.tx.Data.From
	info, err := QueryAddress(sender, context.statedb)
	if err != nil {
		return nil, err
	}

	if info != nil && !info.IsQuit {
		return nil, ErrAlreadyExist
	}

	info = &masternodeInfo{
		IsQuit: false,
	}

	context.statedb.SetData(MasternodeContractAddress, crypto.MustHash(sender), common.SerializePanic(info))

	return nil, nil
}

func queryMasternodeCmd(address []byte, context *Context) ([]byte, error) {
	info, err := getInfo(address, context.statedb)
	if err != nil {
		return nil, err
	}

	if info == nil && !info.IsQuit {
		return ByteTrue, nil
	}

	return ByteFalse, nil
}

func QueryAddress(address common.Address, statedb *state.Statedb) (*masternodeInfo, error) {
	return getInfo(address.Bytes(), statedb)
}

func recallCmd(address []byte, context *Context) ([]byte, error) {
	info, err := getInfo(address, context.statedb)
	if err != nil {
		return nil, err
	}

	if info == nil && !info.IsQuit {
		return nil, ErrNotQuit
	}

	distance := context.BlockHeader.Height - info.QuitBlock
	if info.IsQuit && distance > recallDistanceLimit {
		context.statedb.SetData(MasternodeContractAddress, crypto.MustHash(address), nil)
		context.statedb.SubBalance(MasternodeContractAddress, depositLimit)
		context.statedb.AddBalance(context.tx.Data.From, depositLimit)
	} else {
		return nil, ErrNotEnoughDistance
	}

	return nil, nil
}

func getInfo(address []byte, statedb *state.Statedb) (*masternodeInfo, error) {
	infoBytes := statedb.GetData(MasternodeContractAddress, crypto.MustHash(address))
	if len(infoBytes) == 0 {
		return nil, nil
	}

	var info masternodeInfo
	err := common.Deserialize(infoBytes, &info)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

func saveInfo(address []byte, statedb *state.Statedb, info *masternodeInfo) error {
	buf, err := common.Serialize(info)
	if err != nil {
		return err
	}

	statedb.SetData(MasternodeContractAddress, crypto.MustHash(address), buf)
	return nil
}

func quitCmd(address []byte, context *Context) ([]byte, error) {
	info, err := getInfo(address, context.statedb)
	if err != nil {
		return nil, err
	}

	if info != nil && !info.IsQuit {
		info.IsQuit = true
		info.QuitBlock = context.BlockHeader.Height
		saveInfo(address, context.statedb, info)
	}

	return nil, nil
}
