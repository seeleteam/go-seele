package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/seeleteam/go-seele/node"
	"github.com/stretchr/testify/assert"
)

func getConfig(t *testing.T) *node.Config {
	configFileName := "/testConfig/nodeConfigTest.json"
	currentProjectPath, err := os.Getwd()
	assert.Equal(t, err, nil, "1")
	configFilePath := filepath.Join(currentProjectPath, configFileName)
	accountFilePath := filepath.Join(currentProjectPath, "/testConfig/accounts.json")

	config, err := LoadConfigFromFile(configFilePath, accountFilePath)
	assert.Nil(t, err)

	return config
}

func Test_LoadConfigFromFile(t *testing.T) {
	config := getConfig(t)

	assert.Equal(t, config.BasicConfig.Name, "seele node2", "3")
	assert.Equal(t, config.BasicConfig.Version, "1.0", "4")
	assert.Equal(t, config.BasicConfig.RPCAddr, "0.0.0.0:55028", "5")
	assert.Equal(t, config.BasicConfig.Coinbase, "0x954e4e062eb4bb2dcd93becf4f4e9b1d2d69f131", "6")

	assert.Equal(t, config.HTTPServer.HTTPCors[0], "*", "6")
	assert.Equal(t, config.HTTPServer.HTTPCors[0], "*", "7")
	assert.Equal(t, config.HTTPServer.HTTPAddr, "127.0.0.1:65027", "8")

	assert.Equal(t, config.P2PConfig.ListenAddr, "0.0.0.0:39008", "9")
	assert.Equal(t, config.P2PConfig.NetworkID, "seele", "10")
	assert.Equal(t, len(config.P2PConfig.StaticNodes), 2, "10")
	assert.Equal(t, config.P2PConfig.StaticNodes[0].UDPPort, 39007, "11")
	assert.Equal(t, len(config.P2PConfig.StaticNodes[0].IP), 16, "12")
	assert.Equal(t, config.P2PConfig.StaticNodes[0].TCPPort, 0, "13")

	assert.Equal(t, len(config.SeeleConfig.GenesisConfig.Accounts), 2, "14")
	assert.Equal(t, config.SeeleConfig.GenesisConfig.Difficult, int64(22), "15")
	assert.Equal(t, config.SeeleConfig.GenesisConfig.ShardNumber, uint(1), "16")
}

func Test_CopyConfig(t *testing.T) {
	config := getConfig(t)
	copied := config.Clone()

	assert.Equal(t, config.SeeleConfig.GenesisConfig.ShardNumber, uint(1))
	copied.SeeleConfig.GenesisConfig.ShardNumber = uint(2)
	assert.Equal(t, copied.SeeleConfig.GenesisConfig.ShardNumber, uint(2))
}
