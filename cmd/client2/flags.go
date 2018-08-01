/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"github.com/urfave/cli"
)

var (
	addressValue string
	addressFlag  = cli.StringFlag{
		Name:        "address, a",
		Value:       "127.0.0.1:8027",
		Usage:       "address for client to request",
		Destination: &addressValue,
	}

	accountValue string
	accountFlag  = cli.StringFlag{
		Name:        "account",
		Value:       "",
		Usage:       "account address",
		Destination: &accountValue,
	}

	heightValue int64
	heightFlag  = cli.Int64Flag{
		Name:        "height",
		Value:       0,
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
		Usage:       "from address",
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

	paloadValue string
	paloadFlag  = cli.StringFlag{
		Name:        "payload",
		Value:       "",
		Usage:       "transaction payload info",
		Destination: &paloadValue,
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
)
