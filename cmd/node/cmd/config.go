/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/keystore"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/p2p/discovery"
)

// Config aggregates all configs exposed to users
// Note to add enough comments for every field
type Config struct {
	node.Config

	// private key file of the node for p2p module
	// @TODO need to remove it as keeping private key in memory is very risky
	KeyFile string

	// network id, not used now. @TODO maybe be removed or just use Version
	NetworkID uint64

	// capacity of the transaction pool
	Capacity uint

	// coinbase used by the miner
	Coinbase string

	// static nodes which will be connected to find more nodes when the node starts
	StaticNodes []string

	// core msg interaction uses TCP address and Kademila protocol uses UDP address
	ListenAddr string

	// If IsDebug is true, the log level will be DebugLevel, otherwise it is InfoLevel
	IsDebug bool

	// If PrintLog is true, all logs will be printed in the console, otherwise they will be stored in the file.
	PrintLog bool
}

// GetConfigFromFile unmarshals the config from the given file
func GetConfigFromFile(filepath string) (Config, error) {
	var config Config
	buff, err := ioutil.ReadFile(filepath)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(buff, &config)
	return config, err
}

// LoadConfigFromFile gets node config from the given file
func LoadConfigFromFile(configFile string) (*node.Config, error) {
	config, err := GetConfigFromFile(configFile)
	if err != nil {
		return nil, err
	}

	nodeConfig := new(node.Config)
	nodeConfig.Name = config.Name
	nodeConfig.Version = config.Version
	nodeConfig.RPCAddr = config.RPCAddr
	nodeConfig.HTTPAddr = config.HTTPAddr
	nodeConfig.HTTPCors = config.HTTPCors
	nodeConfig.HTTPWhiteHost = config.HTTPWhiteHost
	nodeConfig.SeeleConfig.Coinbase = common.HexMustToAddres(config.Coinbase)
	nodeConfig.SeeleConfig.NetworkID = config.SeeleConfig.NetworkID
	nodeConfig.SeeleConfig.TxConf.Capacity = config.SeeleConfig.TxConf.Capacity

	nodeConfig.P2P, err = GetP2pConfig(config)
	if err != nil {
		return nil, err
	}

	common.PrintLog = config.PrintLog
	common.IsDebug = config.IsDebug
	nodeConfig.DataDir = filepath.Join(common.GetDefaultDataFolder(), config.DataDir)
	return nodeConfig, nil
}

// GetP2pConfig gets p2p module config from the given config
func GetP2pConfig(config Config) (p2p.Config, error) {
	p2pConfig := p2p.Config{}

	if len(config.StaticNodes) != 0 {
		for _, id := range config.StaticNodes {
			n, err := discovery.NewNodeFromString(id)
			if err != nil {
				return p2p.Config{}, err
			}

			p2pConfig.StaticNodes = append(p2pConfig.StaticNodes, n)
		}
	}

	key, err := keystore.GetKey(config.KeyFile)
	if err != nil {
		return p2pConfig, err
	}

	p2pConfig.PrivateKey = key.PrivateKey
	p2pConfig.ListenAddr = config.ListenAddr
	return p2pConfig, nil
}
