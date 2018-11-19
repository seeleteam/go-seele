/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/contract/system"
	"github.com/seeleteam/go-seele/metrics"
)

var (
	errFormat = "the number of fields in %s is changed, modify default value in system contract subchain"
)

func Test_getConfigFromSubChain(t *testing.T) {
	subChainInfo := system.SubChainInfo{
		Name:              "test",
		Version:           "3.8",
		TokenFullName:     "TestCoin",
		TokenShortName:    "TC",
		TokenAmount:       1000000,
		GenesisDifficulty: 8000,
		GenesisAccounts: map[common.Address]*big.Int{
			common.BytesToAddress([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}): big.NewInt(1000),
			common.BytesToAddress([]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}): big.NewInt(1000),
		},
	}

	config, err := getConfigFromSubChain("seele", &subChainInfo)
	assert.NotEqual(t, err, nil)
	assert.Equal(t, config == nil, true)

	coinbaseValue = "0xe1c54e4b1b3448e382d87e50a427ae319e5ea611"

	config, err = getConfigFromSubChain("seele", &subChainInfo)
	assert.Equal(t, err, nil)
	assert.Equal(t, config != nil, true)
	assert.Equal(t, config.BasicConfig.Coinbase, coinbaseValue)
	assert.Equal(t, config.GenesisConfig.ShardNumber, uint(1))

	reflectBasic := reflect.TypeOf(config.BasicConfig)
	assert.Equalf(t, 6, reflectBasic.NumField(), errFormat, "Node.BasicConfig")

	reflectP2p := reflect.TypeOf(config.P2PConfig)
	assert.Equalf(t, 5, reflectP2p.NumField(), errFormat, "p2p.Config")

	reflectLog := reflect.TypeOf(config.LogConfig)
	assert.Equalf(t, 3, reflectLog.NumField(), errFormat, "comm.LogConfig")

	reflectHTTPServer := reflect.TypeOf(config.HTTPServer)
	assert.Equalf(t, 3, reflectHTTPServer.NumField(), errFormat, "node.HTTPServer")

	reflectWSServer := reflect.TypeOf(config.WSServerConfig)
	assert.Equalf(t, 2, reflectWSServer.NumField(), errFormat, "node.WSServerConfig")

	reflectGenesis := reflect.TypeOf(config.GenesisConfig)
	assert.Equalf(t, 6, reflectGenesis.NumField(), errFormat, "core.GenesisInfo")

	config.MetricsConfig = &metrics.Config{}
	reflectMetrics := reflect.TypeOf(*config.MetricsConfig)
	assert.Equalf(t, 5, reflectMetrics.NumField(), errFormat, "metrics.Config")
}
