package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
)

func Test_LoadConfigFromFile(t *testing.T) {
	assert.Equal(t, common.LogConfig.IsDebug,false)
	assert.Equal(t, common.LogConfig.PrintLog,true)

	configFileName := "/testConfig/nodeConfigTest.json"
	genesisConfigFileName := "/testConfig/genesisTest.json"
	currentProjectPath, err := os.Getwd()
	assert.Equal(t, err, nil)
	configFilePath := filepath.Join(currentProjectPath, configFileName)
	genesisConfigFilePath := filepath.Join(currentProjectPath, genesisConfigFileName)

	config, err := LoadConfigFromFile(configFilePath, genesisConfigFilePath)
	assert.Equal(t, err, nil)
	assert.Equal(t, config.BasicConfig.Name, "seele node2")
	//assert.Equal(t, config.BasicConfig.Capacity, 10000)
	assert.Equal(t, config.BasicConfig.Version, "1.0")
	assert.Equal(t, config.BasicConfig.RPCAddr, "0.0.0.0:55028")
	assert.Equal(t, config.BasicConfig.Coinbase, "0x4dd6881d13ab5152127533c5954e4e062eb4bb2dcd93becf4f4e9b1d2d69f1363eea0395e8e76a2716b033d1e3cc8da2bf24811b1e31a86ac8bcacca4c4b29bd")

	//assert.Equal(t, config.HTTPServer.HTTPCors, "[*]")
	//assert.Equal(t, config.HTTPServer.HTTPCors, "[*]")
	assert.Equal(t, config.HTTPServer.HTTPAddr, "127.0.0.1:65027")

	assert.Equal(t, config.P2PConfig.ListenAddr, "0.0.0.0:39008")
	//assert.Equal(t, config.P2PConfig.ServerPrivateKey, "0x66bfaadbbade123f0dde5c35ec7053f88027ce3ea2f7f0296b99a5e87de6dea7")
	//assert.Equal(t, config.P2PConfig.StaticNodes, "[snode://23ddfb54a488f906cdb9cbd257eac5663a4c74ba25619bb902651602a4491be4ce437907fcc567b31be6746a014931f4670ac116c0010e5beb28b0dce2c6eaad@127.0.0.1:39007]")
	//assert.Equal(t, config.P2PConfig.NetworkID, 1)

	assert.Equal(t, config.LogConfig.IsDebug, true)
	assert.Equal(t, config.LogConfig.PrintLog, true)

	assert.Equal(t, common.LogConfig.IsDebug,true)
	assert.Equal(t, common.LogConfig.PrintLog,true)
}
