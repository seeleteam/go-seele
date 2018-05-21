/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package node

import (
	"crypto/ecdsa"
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log/comm"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/p2p/discovery"
	"github.com/seeleteam/go-seele/seele"
)

// Config is the Configuration of node
type Config struct {
	//Config is the Configuration of log
	LogConfig comm.LogConfig `json:"log"`

	// basic config for Node
	BasicConfig BasicConfig `json:"basic"`

	// The configuration of p2p network
	P2PConfig p2p.Config `json:"p2p"`

	// HttpServer config for http server
	HTTPServer HTTPServer `json:"httpServer"`

	// The SeeleConfig is the configuration to create the seele service.
	SeeleConfig seele.Config
}

// BasicConfig config for Node
type BasicConfig struct {
	// The name of the node
	Name string `json:"name"`

	// The version of the node
	Version string `json:"version"`

	// The file system path of the node, used to store data
	DataDir string `json:"dataDir"`

	// RPCAddr is the address on which to start RPC server.
	RPCAddr string `json:"address"`

	// coinbase used by the miner
	Coinbase string `json:"coinbase"`

	// capacity of the transaction pool
	Capacity uint `json:"capacity"`
}

// HTTPServer config for http server
type HTTPServer struct {
	// The HTTPAddr is the address of HTTP rpc service
	HTTPAddr string `json:"address"`

	// HTTPCors is the Cross-Origin Resource Sharing header to send to requesting
	// clients. Please be aware that CORS is a browser enforced security, it's fully
	// useless for custom HTTP clients.
	HTTPCors []string `json:"crosssorgins"`

	// HTTPHostFilter is the whitelist of hostnames which are allowed on incoming requests.
	HTTPWhiteHost []string `json:"whiteHost"`
}

// GetConfigFromFile unmarshals the config from the given file
func GetConfigFromFile(filepath string) (*Config, error) {
	var config Config
	buff, err := ioutil.ReadFile(filepath)
	if err != nil {
		return &config, err
	}

	err = json.Unmarshal(buff, &config)
	return &config, err
}

// LoadConfigFromFile gets node config from the given file
func LoadConfigFromFile(configFile string, genesisConfigFile string) (*Config, error) {
	config, err := GetConfigFromFile(configFile)
	if err != nil {
		return nil, err
	}

	config.P2PConfig, err = GetP2pConfig(config)
	if err != nil {
		return config, err
	}

	if genesisConfigFile != "" {
		info, err := GetGenesisInfoFromFile(genesisConfigFile)
		if err != nil {
			return nil, err
		}
		config.SeeleConfig.GenesisConfig = info
	}

	config.SeeleConfig.Coinbase = common.HexMustToAddres(config.BasicConfig.Coinbase)
	config.SeeleConfig.NetworkID = config.P2PConfig.OpenConfig.NetworkID
	config.SeeleConfig.TxConf.Capacity = config.BasicConfig.Capacity
	common.LogConfig.PrintLog = config.LogConfig.PrintLog
	common.LogConfig.IsDebug = config.LogConfig.IsDebug
	config.BasicConfig.DataDir = filepath.Join(common.GetDefaultDataFolder(), config.BasicConfig.DataDir)
	return config, nil
}

// GetP2pConfig ResolveStaticNodes from the given config
func GetP2pConfig(config *Config) (p2p.Config, error) {
	if config.P2PConfig.ResolveStaticNodes != nil && config.P2PConfig.PrivateKey != nil {
		return config.P2PConfig, nil
	}

	if config.P2PConfig.ResolveStaticNodes == nil {
		if resolveStaticNodes, err := GetP2pConfigResolveStaticNodes(config); err == nil {
			config.P2PConfig.ResolveStaticNodes = resolveStaticNodes
		} else {
			return config.P2PConfig, err
		}
	}

	if config.P2PConfig.PrivateKey == nil {
		if privateKey, err := GetP2pConfigPrivateKey(config); err == nil {
			config.P2PConfig.PrivateKey = privateKey
		} else {
			return config.P2PConfig, err
		}
	}
	return config.P2PConfig, nil
}

// GetP2pConfigResolveStaticNodes from the given config
func GetP2pConfigResolveStaticNodes(config *Config) ([]*discovery.Node, error) {
	if config.P2PConfig.ResolveStaticNodes != nil {
		return config.P2PConfig.ResolveStaticNodes, nil
	}

	if len(config.P2PConfig.OpenConfig.StaticNodes) != 0 && len(config.P2PConfig.ResolveStaticNodes) == 0 {
		for _, id := range config.P2PConfig.OpenConfig.StaticNodes {
			n, err := discovery.NewNodeFromString(id)
			if err != nil {
				return nil, err
			}

			config.P2PConfig.ResolveStaticNodes = append(config.P2PConfig.ResolveStaticNodes, n)
		}
	}

	return config.P2PConfig.ResolveStaticNodes, nil
}

// GetP2pConfigPrivateKey from the given config
func GetP2pConfigPrivateKey(config *Config) (*ecdsa.PrivateKey, error) {
	if config.P2PConfig.PrivateKey != nil {
		return config.P2PConfig.PrivateKey, nil
	}

	key, err := crypto.LoadECDSAFromString(config.P2PConfig.OpenConfig.ServerPrivateKey)
	if err != nil {
		return nil, err
	}
	return key, err
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
