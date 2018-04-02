/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"io/ioutil"
	"encoding/json"
)

// aggregate all configs here that exposed to users
// Note add enough comments for every parameter
type Config struct {
	Name string
	Version string
	RPCAddr string
	ECDSAKey string
	NetworkID uint64
	Coinbase string
	Capacity int
}

func GetConfigFromFile(filepath string) (Config, error) {
	var config Config
	buff, err := ioutil.ReadFile(filepath)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(buff, &config)
	return config, err
}