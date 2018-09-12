/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"errors"

	"github.com/seeleteam/go-seele/rpc2"
)

type handler func(client *rpc.Client) (interface{}, interface{}, error)

var (
	errInvalidCommand    = errors.New("invalid command")
	errInvalidSubcommand = errors.New("invalid subcommand")

	systemContract = map[string]map[string]handler{
		"htlc": map[string]handler{
			"create":   createHTLC,
			"withdraw": withdraw,
			"refund":   refund,
			"get":      getHTLC,
		},
		"domain": map[string]handler{
			"register":     registerDomainName,
			"getregistrar": domainNameRegister,
		},
	}
)
