/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package node

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/p2p/discovery"
	"github.com/seeleteam/go-seele/seele"
)

// Note to add enough comments for every field
type Config struct {
	// ServerPrivateKey private key for p2p module, do not use it as any accounts
	ServerPrivateKey string `json:"serverPrivateKey"`

	// network id, not used now. @TODO maybe be removed or just use Version
	NetworkID uint64 `json:"networkID"`

	// capacity of the transaction pool
	Capacity uint `json:"capacity"`

	// coinbase used by the miner
	Coinbase string `json:"coinbase"`

	// static nodes which will be connected to find more nodes when the node starts
	StaticNodes []string `json:"staticNodes"`

	// core msg interaction uses TCP address and Kademila protocol uses UDP address
	ListenAddr string `json:"listenAddr"`

	// If IsDebug is true, the log level will be DebugLevel, otherwise it is InfoLevel
	IsDebug bool `json:"isDebug"`

	// If PrintLog is true, all logs will be printed in the console, otherwise they will be stored in the file.
	PrintLog bool `json:"printLog"`

	// basic config for Node
	Basic Basic `json:"basic"`

	// The configuration of p2p network
	P2P p2p.Config

	// HttpServer config for http server
	HTTPServer HTTPServer `json:"HTTPServer"`

	// The SeeleConfig is the configuration to create the seele service.
	SeeleConfig seele.Config
}

// Basic config for Node
type Basic struct {
	// The name of the node
	Name string `json:"name"`

	// The version of the node
	Version string `json:"version"`

	// The file system path of the node, used to store data
	DataDir string `json:"dataDir"`

	// RPCAddr is the address on which to start RPC server.
	RPCAddr string `json:"rpcAddr"`
}

// HTTPServer config for http server
type HTTPServer struct {
	// The HTTPAddr is the address of HTTP rpc service
	HTTPAddr string `json:"httpAddr"`

	// HTTPCors is the Cross-Origin Resource Sharing header to send to requesting
	// clients. Please be aware that CORS is a browser enforced security, it's fully
	// useless for custom HTTP clients.
	HTTPCors []string `json:"httpCors"`

	// HTTPHostFilter is the whitelist of hostnames which are allowed on incoming requests.
	HTTPWhiteHost []string `json:"httpWhiteHost"`
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
func LoadConfigFromFile(configFile string, genesisConfigFile string) (*Config, error) {
	config, err := GetConfigFromFile(configFile)
	if err != nil {
		return nil, err
	}

	nodeConfig := new(Config)
	nodeConfig.Basic.Name = config.Basic.Name
	nodeConfig.Basic.Version = config.Basic.Version
	nodeConfig.Basic.RPCAddr = config.Basic.RPCAddr
	nodeConfig.HTTPServer.HTTPAddr = config.HTTPServer.HTTPAddr
	nodeConfig.HTTPServer.HTTPCors = config.HTTPServer.HTTPCors
	nodeConfig.HTTPServer.HTTPWhiteHost = config.HTTPServer.HTTPWhiteHost

	nodeConfig.P2P, err = GetP2pConfig(config)
	if err != nil {
		return nil, err
	}

	if genesisConfigFile != "" {
		info, err := GetGenesisInfoFromFile(genesisConfigFile)
		if err != nil {
			return nil, err
		}
		nodeConfig.SeeleConfig.GenesisConfig = info
	}

	nodeConfig.SeeleConfig.Coinbase = common.HexMustToAddres(config.Coinbase)
	nodeConfig.SeeleConfig.NetworkID = config.NetworkID
	nodeConfig.SeeleConfig.TxConf.Capacity = config.Capacity

	common.PrintLog = config.PrintLog
	common.IsDebug = config.IsDebug
	nodeConfig.Basic.DataDir = filepath.Join(common.GetDefaultDataFolder(), config.Basic.DataDir)
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

	key, err := crypto.LoadECDSAFromString(config.ServerPrivateKey)
	if err != nil {
		return p2pConfig, err
	}

	p2pConfig.PrivateKey = key
	p2pConfig.ListenAddr = config.ListenAddr
	return p2pConfig, nil
}

// GetGenesisInfoFromFile get genesis info from a specific file
func GetGenesisInfoFromFile(filepath string) (core.GenesisInfo, error) {
	var info core.GenesisInfo
	buff, err := ioutil.ReadFile(filepath)
	if err != nil {
		return info, err
	}

	err = json.Unmarshal(buff, &info)
	return info, err
}
