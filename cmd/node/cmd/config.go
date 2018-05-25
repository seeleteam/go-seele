/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"crypto/ecdsa"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log/comm"
	"github.com/seeleteam/go-seele/metrics"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/rpc"
)

// Config is the Configuration of node
type Config struct {
	//Config is the Configuration of log
	LogConfig comm.LogConfig `json:"log"`

	// basic config for Node
	BasicConfig node.BasicConfig `json:"basic"`

	// The configuration of p2p network
	P2PConfig p2p.Config `json:"p2p"`

	// HttpServer config for http server
	HTTPServer node.HTTPServer `json:"httpServer"`

	// The configuration of websocket rpc service
	WSServerConfig rpc.WSServerConfig `json:"wsserver"`

	// metrics config info
	MetricsConfig metrics.Config `json:"metrics"`
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

	config.P2PConfig, err = GetP2pConfig(cmdConfig)
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
	config.SeeleConfig.TxConf = *core.DefaultTxPoolConfig()
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
		P2PConfig:      cmdConfig.P2PConfig,
		SeeleConfig:    node.SeeleConfig{},
		MetricsConfig:  cmdConfig.MetricsConfig,
	}
	return config
}

// GetP2pConfig get P2PConfig from the given config
func GetP2pConfig(cmdConfig *Config) (p2p.Config, error) {
	if cmdConfig.P2PConfig.PrivateKey == nil {
		if privateKey, err := GetP2pConfigPrivateKey(cmdConfig); err == nil {
			cmdConfig.P2PConfig.PrivateKey = privateKey
		} else {
			return cmdConfig.P2PConfig, err
		}
	}
	return cmdConfig.P2PConfig, nil
}

// GetP2pConfigPrivateKey get privateKey from the given config
func GetP2pConfigPrivateKey(cmdConfig *Config) (*ecdsa.PrivateKey, error) {
	if cmdConfig.P2PConfig.PrivateKey != nil {
		return cmdConfig.P2PConfig.PrivateKey, nil
	}

	key, err := crypto.LoadECDSAFromString(cmdConfig.P2PConfig.SubPrivateKey)
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
