/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import "github.com/urfave/cli"

var (
	addressValue string
	addressFlag  = cli.StringFlag{
		Name:        "address, a",
		Value:       "127.0.0.1:8027",
		Usage:       "address for client to request",
		Destination: &addressValue,
	}
)
