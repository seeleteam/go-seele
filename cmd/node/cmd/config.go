/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"crypto/ecdsa"
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log/comm"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/p2p/discovery"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/seeleteam/go-seele/seele"
)

// Config is the Configuration of node
type Config struct {
	//Config is the Configuration of log
	LogConfig comm.LogConfig `json:"log"`

	// basic config for Node
	BasicConfig node.BasicConfig `json:"basic"`

	// The configuration of p2p network
	P2PConfig p2p.P2PConfig `json:"p2p"`

	// HttpServer config for http server
	HTTPServer node.HTTPServer `json:"httpServer"`

	// The SeeleConfig is the configuration to create the seele service.
	SeeleConfig seele.Config

	// The configuration of websocket rpc service
	WSServerConfig rpc.WSServerConfig `json:"wsserver"`
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
func LoadConfigFromFile(configFile string, genesisConfigFile string) (*node.Config, error) {
	cmdConfig, err := GetConfigFromFile(configFile)
	if err != nil {
		return nil, err
	}

	config := CopyConfig(cmdConfig)

	config.P2PConfig, err = GetP2pConfig(config, cmdConfig)
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
	config.SeeleConfig.NetworkID = config.P2PConfig.NetworkID
	config.SeeleConfig.TxConf.Capacity = config.BasicConfig.Capacity
	common.LogConfig.PrintLog = config.LogConfig.PrintLog
	common.LogConfig.IsDebug = config.LogConfig.IsDebug
	config.BasicConfig.DataDir = filepath.Join(common.GetDefaultDataFolder(), config.BasicConfig.DataDir)
	return config, nil
}

// CopyConfig copy Config from the given config
func CopyConfig(cmdConfig *Config) *node.Config {
	config := &node.Config{
		BasicConfig:    cmdConfig.BasicConfig,
		LogConfig:      cmdConfig.LogConfig,
		HTTPServer:     cmdConfig.HTTPServer,
		WSServerConfig: cmdConfig.WSServerConfig,
		P2PConfig:      p2p.Config{ListenAddr: cmdConfig.P2PConfig.ListenAddr, NetworkID: cmdConfig.P2PConfig.NetworkID},
		SeeleConfig:    seele.Config{},
	}
	return config
}

// GetP2pConfig get P2PConfig from the given config
func GetP2pConfig(config *node.Config, cmdConfig *Config) (p2p.Config, error) {
	if config.P2PConfig.ResolveStaticNodes != nil && config.P2PConfig.PrivateKey != nil {
		return config.P2PConfig, nil
	}

	if config.P2PConfig.ResolveStaticNodes == nil {
		if resolveStaticNodes, err := GetP2pConfigResolveStaticNodes(config, cmdConfig); err == nil {
			config.P2PConfig.ResolveStaticNodes = resolveStaticNodes
		} else {
			return config.P2PConfig, err
		}
	}

	if config.P2PConfig.PrivateKey == nil {
		if privateKey, err := GetP2pConfigPrivateKey(config, cmdConfig); err == nil {
			config.P2PConfig.PrivateKey = privateKey
		} else {
			return config.P2PConfig, err
		}
	}
	return config.P2PConfig, nil
}

// GetP2pConfigResolveStaticNodes get ResolveStaticNodes from the given config
func GetP2pConfigResolveStaticNodes(config *node.Config, cmdConfig *Config) ([]*discovery.Node, error) {
	if config.P2PConfig.ResolveStaticNodes != nil {
		return config.P2PConfig.ResolveStaticNodes, nil
	}

	if len(cmdConfig.P2PConfig.StaticNodes) != 0 && len(config.P2PConfig.ResolveStaticNodes) == 0 {
		for _, id := range cmdConfig.P2PConfig.StaticNodes {
			n, err := discovery.NewNodeFromString(id)
			if err != nil {
				return nil, err
			}

			config.P2PConfig.ResolveStaticNodes = append(config.P2PConfig.ResolveStaticNodes, n)
		}
	}

	return config.P2PConfig.ResolveStaticNodes, nil
}

// GetP2pConfigPrivateKey get privateKey from the given config
func GetP2pConfigPrivateKey(config *node.Config, cmdConfig *Config) (*ecdsa.PrivateKey, error) {
	if config.P2PConfig.PrivateKey != nil {
		return config.P2PConfig.PrivateKey, nil
	}

	key, err := crypto.LoadECDSAFromString(cmdConfig.P2PConfig.ServerPrivateKey)
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
