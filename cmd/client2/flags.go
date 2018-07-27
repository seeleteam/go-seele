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

	heightValue uint64
	heightFlag  = cli.Uint64Flag{
		Name:        "height, h",
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
)
