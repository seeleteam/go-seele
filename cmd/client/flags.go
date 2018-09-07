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
)
