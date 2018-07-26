/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package metrics

import (
	"fmt"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log"
	influxdb "github.com/seeleteam/go-seele/metrics/go-metrics-influxdb"
)

var MetricsWriteBlockMeter = metrics.GetOrRegisterMeter("core.blockchain.writeBlock.time", nil)

// Config infos for influxdb
type Config struct {
	Addr     string        `json:"address"`
	Database string        `json:"database"`
	Username string        `json:"username"`
	Password string        `json:"password"`
	Duration time.Duration `json:"duration"`
}

// StartMetricsWithConfig start recording metrics with configure
func StartMetricsWithConfig(conf *Config, log *log.SeeleLog, name, version string, networkID uint64, coinBase common.Address) {
	if conf == nil {
		log.Error("failed to start the metrics: the config of metrics is null")
		return
	}

	StartMetrics(
		time.Second*conf.Duration,
		conf.Addr,
		conf.Database,
		conf.Username,
		conf.Password,
		name,
		networkID,
		version,
		coinBase,
		log,
	)
}

// StartMetrics start recording metrics
func StartMetrics(
	d time.Duration, // duration to send metrics datas
	address string, // remote influxdb address
	database string, // database 'influxdb'
	username string, // database username
	password string, // database password
	nodeName string, // name of the node
	networkID uint64,
	version string,
	coinBase common.Address,
	log *log.SeeleLog) {
	log.Info("Start metrics!")

	go influxdb.InfluxDBWithTags(
		metrics.DefaultRegistry,
		d,
		address,
		database,
		username,
		password,
		map[string]string{
			"nodename":  nodeName,
			"networkid": fmt.Sprint(networkID),
			"version":   version,
			"coinbase":  coinBase.ToHex(),
			"shardid":   fmt.Sprint(common.LocalShardNumber),
		},
		log,
	)

	go collectRuntimeMetrics()
}
