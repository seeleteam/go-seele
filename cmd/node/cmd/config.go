/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"runtime"

	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log/comm"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/p2p"
)

// GetConfigFromFile unmarshals the config from the given file
func GetConfigFromFile(filepath string) (*util.Config, error) {
	var config util.Config
	buff, err := ioutil.ReadFile(filepath)
	if err != nil {
		return &config, err
	}

	err = json.Unmarshal(buff, &config)
	return &config, err
}

// LoadConfigFromFile gets node config from the given file
func LoadConfigFromFile(configFile string, accounts string) (*node.Config, error) {
	cmdConfig, err := GetConfigFromFile(configFile)
	if err != nil {
		return nil, err
	}

	if cmdConfig.GenesisConfig.CreateTimestamp == nil {
		return nil, errors.New("Failed to get genesis timestamp")
	}

	cmdConfig.GenesisConfig.Accounts, err = LoadAccountConfig(accounts)
	if err != nil {
		return nil, err
	}

	config := CopyConfig(cmdConfig)
	convertIPCServerPath(cmdConfig, config)

	config.P2PConfig, err = GetP2pConfig(cmdConfig)
	if err != nil {
		return config, err
	}

	if len(config.BasicConfig.Coinbase) > 0 {
		config.SeeleConfig.Coinbase = common.HexMustToAddres(config.BasicConfig.Coinbase)
	}

	config.SeeleConfig.TxConf = *core.DefaultTxPoolConfig()
	config.SeeleConfig.GenesisConfig = cmdConfig.GenesisConfig
	comm.LogConfiguration.PrintLog = config.LogConfig.PrintLog
	comm.LogConfiguration.IsDebug = config.LogConfig.IsDebug
	comm.LogConfiguration.DataDir = config.BasicConfig.DataDir
	config.BasicConfig.DataDir = filepath.Join(common.GetDefaultDataFolder(), config.BasicConfig.DataDir)
	return config, nil
}

// convertIPCServerPath convert the config to the real path
func convertIPCServerPath(cmdConfig *util.Config, config *node.Config) {
	if cmdConfig.Ipcconfig.PipeName == "" {
		config.IpcConfig.PipeName = common.GetDefaultIPCPath()
	} else if runtime.GOOS == "windows" {
		config.IpcConfig.PipeName = common.WindowsPipeDir + cmdConfig.Ipcconfig.PipeName
	} else {
		config.IpcConfig.PipeName = filepath.Join(common.GetDefaultDataFolder(), cmdConfig.Ipcconfig.PipeName)
	}
}

// CopyConfig copy Config from the given config
func CopyConfig(cmdConfig *util.Config) *node.Config {
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
func GetP2pConfig(cmdConfig *util.Config) (p2p.Config, error) {
	if cmdConfig.P2PConfig.PrivateKey == nil {
		key, err := crypto.LoadECDSAFromString(cmdConfig.P2PConfig.SubPrivateKey) // GetP2pConfigPrivateKey get privateKey from the given config
		if err != nil {
			return cmdConfig.P2PConfig, err
		}
		cmdConfig.P2PConfig.PrivateKey = key
	}
	return cmdConfig.P2PConfig, nil
}

func LoadAccountConfig(account string) (map[common.Address]*big.Int, error) {
	result := make(map[common.Address]*big.Int)
	if account == "" {
		return result, nil
	}

	buff, err := ioutil.ReadFile(account)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(buff, &result)
	return result, err
}
