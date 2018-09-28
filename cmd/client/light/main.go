/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package main

import (
	"log"
	"os"

	"github.com/seeleteam/go-seele/cmd/client/cmd"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "light node client"
	app.Usage = "interact with node process"
	app.HideVersion = true
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "seeleteam",
			Email: "dev@seelenet.com",
		},
	}

	cmd.AddCommands(app, false)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
