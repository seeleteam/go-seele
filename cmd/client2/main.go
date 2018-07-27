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
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
