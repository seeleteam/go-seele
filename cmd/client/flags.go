/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/urfave/cli"
)

type rpcFlag interface {
	getValue() (interface{}, error)
}

type seeleAddressFlag struct {
	cli.StringFlag
}

func (flag seeleAddressFlag) getValue() (interface{}, error) {
	if val := *flag.Destination; len(val) > 0 {
		return common.HexToAddress(val)
	}

	return common.EmptyAddress, nil
}

var (
	addressValue string
	addressFlag  = cli.StringFlag{
		Name:        "address, a",
		Value:       "127.0.0.1:8027",
		Usage:       "address for client to request",
		Destination: &addressValue,
	}

	accountValue string
	accountFlag  = seeleAddressFlag{
		StringFlag: cli.StringFlag{
			Name:        "account",
			Value:       "",
			Usage:       "account address",
			Destination: &accountValue,
		},
	}

	heightValue int64
	heightFlag  = cli.Int64Flag{
		Name:        "height",
		Value:       -1,
		Usage:       "block height",
		Destination: &heightValue,
	}

	fulltxValue bool
	fulltxFlag  = cli.BoolFlag{
		Name:        "fulltx, f",
		Usage:       "whether print full transaction info",
		Destination: &fulltxValue,
	}

	hashValue string
	hashFlag  = cli.StringFlag{
		Name:        "hash",
		Usage:       "hash value in hex",
		Destination: &hashValue,
	}

	fromValue string
	fromFlag  = cli.StringFlag{
		Name:        "from",
		Usage:       "key file of the sender",
		Destination: &fromValue,
	}

	toValue string
	toFlag  = cli.StringFlag{
		Name:        "to",
		Usage:       "to address",
		Destination: &toValue,
	}

	amountValue string
	amountFlag  = cli.StringFlag{
		Name:        "amount",
		Usage:       "amount value, unit is fan",
		Destination: &amountValue,
	}

	payloadValue string
	payloadFlag  = cli.StringFlag{
		Name:        "payload",
		Value:       "",
		Usage:       "transaction payload info",
		Destination: &payloadValue,
	}

	feeValue string
	feeFlag  = cli.StringFlag{
		Name:        "fee",
		Usage:       "transaction fee",
		Destination: &feeValue,
	}

	nonceValue uint64
	nonceFlag  = cli.Uint64Flag{
		Name:        "nonce",
		Value:       0,
		Usage:       "transaction nonce",
		Destination: &nonceValue,
	}

	contractValue string
	contractFlag  = cli.StringFlag{
		Name:        "contract",
		Usage:       "contract code in hex",
		Destination: &contractValue,
	}

	topicValue string
	topicFlag  = cli.StringFlag{
		Name:        "topic",
		Usage:       "topic",
		Destination: &topicValue,
	}

	threadsValue uint
	threadsFlag  = cli.UintFlag{
		Name:        "threads",
		Usage:       "miner threads",
		Destination: &threadsValue,
	}

	coinbaseValue string
	coinbaseFlag  = cli.StringFlag{
		Name:        "coinbase",
		Usage:       "miner coinbase in hex",
		Destination: &coinbaseValue,
	}

	indexValue uint
	indexFlag  = cli.UintFlag{
		Name:        "index",
		Usage:       "transaction index, start with 0",
		Value:       0,
		Destination: &indexValue,
	}

	privateKeyValue string
	privateKeyFlag  = cli.StringFlag{
		Name:        "privatekey",
		Usage:       "private key for account",
		Destination: &privateKeyValue,
	}

	fileNameValue string
	fileNameFlag  = cli.StringFlag{
		Name:        "file",
		Value:       ".keystore",
		Usage:       "key store file name",
		Destination: &fileNameValue,
	}

	shardValue uint
	shardFlag  = cli.UintFlag{
		Name:        "shard",
		Usage:       "shard number",
		Destination: &shardValue,
	}

	gcBeforeDump     bool
	gcBeforeDumpFlag = cli.BoolFlag{
		Name:        "gc",
		Usage:       "GC before heap dump, defualt false",
		Destination: &gcBeforeDump,
	}

	dumpFileValue string
	dumpFileFlag  = cli.StringFlag{
		Name:        "file",
		Value:       "heap.dump",
		Usage:       "heap dump file name, defualt heap.dump",
		Destination: &dumpFileValue,
	}

	timeLockValue int64
	timeLockFlag  = cli.Int64Flag{
		Name:        "time",
		Usage:       "time lock in the HTLC",
		Destination: &timeLockValue,
	}

	preimageValue string
	preimageFlag  = cli.StringFlag{
		Name:        "preimage",
		Usage:       "preimage of hash in the HTLC",
		Destination: &preimageValue,
	}

	domainNameValue string
	domainNameFlag  = cli.StringFlag{
		Name:        "name",
		Usage:       "domain name",
		Destination: &domainNameValue,
	}

	subChainNameVale string
	subChainNameFlag = cli.StringFlag{
		Name:        "name",
		Usage:       "subchain name",
		Destination: &subChainNameVale,
	}

	subChainVersionValue string
	subChainVersionFlag  = cli.StringFlag{
		Name:        "version, v",
		Usage:       "subchain version",
		Destination: &subChainVersionValue,
	}

	subChainTokenFullNameValue string
	subChainTokenFullNameFlag  = cli.StringFlag{
		Name:        "fullname",
		Usage:       "subchain token full name",
		Destination: &subChainTokenFullNameValue,
	}

	subChainTokenShortNameValue string
	subChainTokenShortNameFlag  = cli.StringFlag{
		Name:        "shortname",
		Usage:       "subchain token short name",
		Destination: &subChainTokenShortNameValue,
	}

	subChainTokenAmountValue uint64
	subChainTokenAmountFlag  = cli.Uint64Flag{
		Name:        "amount",
		Usage:       "subchain token amount",
		Destination: &subChainTokenAmountValue,
	}

	subChainStaticNodesValue cli.StringSlice
	subChainStaticNodesFlag  = cli.StringSliceFlag{
		Name:  "nodes, n",
		Usage: "subchain token static nodes, for example: -n address:port",
		Value: &subChainStaticNodesValue,
	}

	subChainGenesisAccountsValue cli.StringSlice
	subChainGenesisAccountsFlag  = cli.StringSliceFlag{
		Name:  "accounts",
		Usage: "subchain token genesis accounts, for example: -a address:amount",
		Value: &subChainGenesisAccountsValue,
	}

	subChainGenesisDifficultyValue uint64
	subChainGenesisDifficultyFlag  = cli.Uint64Flag{
		Name:        "difficulty, d",
		Usage:       "subchain token difficulty",
		Destination: &subChainGenesisDifficultyValue,
	}
)
