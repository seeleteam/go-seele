/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/seeleteam/go-seele/rpc2"
	"github.com/seeleteam/go-seele/seele"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()

	getinfoAction := func(c *cli.Context) error {
		client, err := rpc.DialTCP(context.Background(), addressValue)
		if err != nil {
			return err
		}

		var info seele.MinerInfo
		err = client.Call(&info, "seele_getInfo")
		if err != nil {
			return err
		}

		result, err := json.MarshalIndent(info, "", "\t")
		fmt.Printf("%s\n", result)

		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:  "getinfo",
			Usage: "get miner info",
			Flags: []cli.Flag{
				addressFlag,
			},
			Action: getinfoAction,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
