/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

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
		Usage:       "block height or current block height for negative value",
		Destination: &heightValue,
	}

	trialValue string
	trialFlag  = cli.StringFlag{
		Name:        "trial, t",
		Value:       "false",
		Usage:       "trial for transanction or contract, value of false is to send tx or contract, and true is to call",
		Destination: &trialValue,
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

	priceValue string
	priceFlag  = cli.StringFlag{
		Name:        "price",
		Value:       "10",
		Usage:       "transaction gas price in Fan",
		Destination: &priceValue,
	}

	gasLimitValue uint64
	gasLimitFlag  = cli.Uint64Flag{
		Name:        "gas",
		Value:       200000,
		Usage:       "maximum gas for transaction",
		Destination: &gasLimitValue,
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
		Usage:       "GC before heap dump, default false",
		Destination: &gcBeforeDump,
	}

	dumpFileValue string
	dumpFileFlag  = cli.StringFlag{
		Name:        "file",
		Value:       "heap.dump",
		Usage:       "heap dump file name, default heap.dump",
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

	nameValue string
	nameFlag  = cli.StringFlag{
		Name:        "name",
		Usage:       "domain or subchain name",
		Destination: &nameValue,
	}

	subChainJSONFileVale string
	subChainJSONFileFlag = cli.StringFlag{
		Name:        "file",
		Usage:       "subchain json file path",
		Destination: &subChainJSONFileVale,
	}

	outPutValue string
	outPutFlag  = cli.StringFlag{
		Name:        "output,o",
		Usage:       "subchain config file path",
		Destination: &outPutValue,
	}

	staticNodesValue cli.StringSlice
	staticNodesFlag  = cli.StringSliceFlag{
		Name:  "node, n",
		Usage: "subchain static node, for example:-n address:port -n address:prot",
		Value: &staticNodesValue,
	}

	algorithmValue string
	algorithmFlag  = cli.StringFlag{
		Name:        "algorithm",
		Usage:       "miner algorithm",
		Value:       "sha256",
		Destination: &algorithmValue,
	}
)

// GeneratePayload
var (
	abiFile     string
	abiFileFlag = cli.StringFlag{
		Name:        "abi",
		Usage:       "the abi file of contract",
		Destination: &abiFile,
	}

	methodName     string
	methodNameFlag = cli.StringFlag{
		Name:        "method",
		Usage:       "the method name of contract",
		Destination: &methodName,
	}

	eventName     string
	eventNameFlag = cli.StringFlag{
		Name:        "event",
		Usage:       "the event name of contract",
		Destination: &eventName,
	}

	// args     []interface{}
	argsFlag = cli.StringSliceFlag{
		Name:  "args",
		Usage: "the parameters of contract method",
	}
)
