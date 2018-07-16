package metrics

import (
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
)

const (
	TestName      = "seele node1"
	TestVersion   = "1.0"
	TestNetworkID = 1
)

var TestCoinbase = crypto.MustGenerateShardAddress(1)

func getTmpConfig() *Config {
	return &Config{
		Addr:     "127.0.0.1:8087",
		Duration: 10,
		Database: "influxdb",
		Username: "test",
		Password: "test123",
	}
}

func Test_StartMetricsWithConfig(t *testing.T) {
	nCfg := getTmpConfig()
	slog := log.GetLogger("seele", common.LogConfig.PrintLog)
	StartMetricsWithConfig(
		nCfg,
		slog,
		TestName,
		TestVersion,
		TestNetworkID,
		*TestCoinbase,
	)
}
