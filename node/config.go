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
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/p2p/discovery"
	"github.com/seeleteam/go-seele/seele"
)

// Note to add enough comments for every field
type Config struct {
	//Config is the Configuration of log
	Log log.Config

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

	// coinbase used by the miner
	Coinbase string `json:"coinbase"`
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

	nodeConfig.SeeleConfig.Coinbase = common.HexMustToAddres(config.Basic.Coinbase)
	nodeConfig.SeeleConfig.NetworkID = config.P2P.NetworkID
	nodeConfig.SeeleConfig.TxConf.Capacity = config.P2P.Capacity

	common.PrintLog = config.Log.PrintLog
	common.IsDebug = config.Log.IsDebug
	nodeConfig.Basic.DataDir = filepath.Join(common.GetDefaultDataFolder(), config.Basic.DataDir)
	return nodeConfig, nil
}

// GetP2pConfig gets p2p module config from the given config
func GetP2pConfig(config Config) (p2p.Config, error) {
	p2pConfig := p2p.Config{}

	if len(config.P2P.StaticNodes) != 0 {
		for _, id := range config.P2P.StaticNodes {
			n, err := discovery.NewNodeFromString(id)
			if err != nil {
				return p2p.Config{}, err
			}

			p2pConfig.ResolveStaticNodes = append(p2pConfig.ResolveStaticNodes, n)
		}
	}

	key, err := crypto.LoadECDSAFromString(config.P2P.ServerPrivateKey)
	if err != nil {
		return p2pConfig, err
	}

	p2pConfig.PrivateKey = key
	p2pConfig.ListenAddr = config.P2P.ListenAddr
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
