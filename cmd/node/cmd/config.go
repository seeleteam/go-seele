/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"encoding/json"
	"io/ioutil"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/p2p/discovery"
)

// Config aggregate all configs here that exposed to users
// Note add enough comments for every parameter
type Config struct {
	// The name of the node
	Name string

	// The version of the node
	Version string

	// The folder used to store block data
	DataDir string

	// JSON API address
	RPCAddr string

	// private key of the node for p2p module
	// @TODO need to remove it as keep private key in memory is very risk
	ECDSAKey string

	// network id, not using for now, @TODO maybe remove or just use Version
	NetworkID uint64

	// coinbase that miner use
	Coinbase string

	// capacity of trasaction pool
	Capacity uint

	// static nodes when node start, it will connect with them to find more nodes
	StaticNodes []string

	// Kademila protocol use UDP address
	KadAddr string

	// core msg interaction TCP address
	ListenAddr string
}

// GetConfigFromFile unmarshal config from a file
func GetConfigFromFile(filepath string) (Config, error) {
	var config Config
	buff, err := ioutil.ReadFile(filepath)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(buff, &config)
	return config, err
}

// GetNodeConfigFromFile get node config from a file
func GetNodeConfigFromFile(configFile string) (*node.Config, error) {
	config, err := GetConfigFromFile(configFile)
	if err != nil {
		return nil, err
	}

	nodeConfig := new(node.Config)
	nodeConfig.Name = config.Name
	nodeConfig.Version = config.Version
	nodeConfig.RPCAddr = config.RPCAddr
	nodeConfig.SeeleConfig.Coinbase = common.HexMustToAddres(config.Coinbase)
	nodeConfig.SeeleConfig.NetworkID = config.NetworkID
	nodeConfig.SeeleConfig.TxConf.Capacity = config.Capacity

	nodeConfig.P2P, err = GetP2pConfig(config)
	if err != nil {
		return nil, err
	}

	nodeConfig.DataDir = config.DataDir
	return nodeConfig, nil
}

// GetP2pConfig get p2p module config from config
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

	p2pConfig.ECDSAKey = config.ECDSAKey
	p2pConfig.KadAddr = config.KadAddr
	p2pConfig.ListenAddr = config.ListenAddr
	return p2pConfig, nil
}
