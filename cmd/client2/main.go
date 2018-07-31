/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"log"
	"os"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()

	app.Commands = []cli.Command{
		{
			Name:  "getinfo",
			Usage: "get miner info",
			Flags: []cli.Flag{
				addressFlag,
			},
			Action: RPCAction(GetInfoAction),
		},
		{
			Name:  "getbalance",
			Usage: "get balance info",
			Flags: []cli.Flag{
				addressFlag,
				accountFlag,
			},
			Action: RPCAction(GetBalanceAction),
		},
		{
			Name:  "sendtx",
			Usage: "send transaction to node",
			Flags: []cli.Flag{
				addressFlag,
				fromFlag,
				toFlag,
				amountFlag,
				feeFlag,
				paloadFlag,
				nonceFlag,
			},
			Action: RPCAction(AddTxAction),
		},
		{
			Name:  "getnonce",
			Usage: "get account nonce",
			Flags: []cli.Flag{
				addressFlag,
				accountFlag,
			},
			Action: RPCAction(GetAccountNonceAction),
		},
		{
			Name:  "getblockheight",
			Usage: "get block height",
			Flags: []cli.Flag{
				addressFlag,
			},
			Action: RPCAction(GetBlockHeightAction),
		},
		{
			Name:  "getblockbyheight",
			Usage: "get block by height",
			Flags: []cli.Flag{
				addressFlag,
				heightFlag,
				fulltxFlag,
			},
			Action: RPCAction(GetBlockByHeightAction),
		},
		{
			Name:  "getblockbyhash",
			Usage: "get block by hash",
			Flags: []cli.Flag{
				addressFlag,
				hashFlag,
				fulltxFlag,
			},
			Action: RPCAction(GetBlockByHashAction),
		},
		{
			Name:  "getlogs",
			Usage: "get logs",
			Flags: []cli.Flag{
				addressFlag,
				heightFlag,
				contractFlag,
				topicFlag,
			},
			Action: RPCAction(GetLogsAction),
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
