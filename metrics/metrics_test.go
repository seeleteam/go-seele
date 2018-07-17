package metrics

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
)

const (
	TestName      = "seele node1"
	TestVersion   = "1.0"
	TestNetworkID = 1
)

var (
	TestCoinbase = crypto.MustGenerateShardAddress(1)
	slog         = log.GetLogger("seele", common.LogConfig.PrintLog)
	address      = "127.0.0.1:8086"
	result       = new(string)
)

func getTmpConfig() *Config {
	return &Config{
		Addr:     address,
		Duration: 1,
		Database: "influxdb",
		Username: "test",
		Password: "test123",
	}
}

// saveResult will Save the data
func saveResult(w http.ResponseWriter, r *http.Request) {
	fmt.Println("path", r.URL.Path)
	s, _ := ioutil.ReadAll(r.Body)
	*result = *result + string(s)
}

// influxdbSimulate simulate the influxdb server
func influxdbSimulate() {
	http.HandleFunc("/write", saveResult)
	err := http.ListenAndServe(address, nil)
	if err != nil {
		slog.Fatal("ListenAndServe: ", err)
	}
}

func markTest() {
	testGauge := metrics.GetOrRegisterGauge("test.Gauge", metrics.DefaultRegistry)
	testCounter := metrics.GetOrRegisterCounter("test.Counter", metrics.DefaultRegistry)
	testMeter := metrics.GetOrRegisterMeter("test.Meter", metrics.DefaultRegistry)
	testGaugeFloat64 := metrics.GetOrRegisterGaugeFloat64("test.GaugeFloat64", metrics.DefaultRegistry)
	testGaugeHistogram := metrics.GetOrRegisterHistogram("test.Histogram", metrics.DefaultRegistry, metrics.NewUniformSample(6))
	testGaugeTimer := metrics.GetOrRegisterTimer("test.Timer", metrics.DefaultRegistry)

	testGauge.Update(6)
	testCounter.Count()
	testMeter.Mark(2)
	testGaugeFloat64.Update(6.6)
	testGaugeHistogram.Update(6)
	testGaugeTimer.Update(time.Microsecond)
}

func Test_StartMetrics(t *testing.T) {
	go influxdbSimulate()

	nCfg := getTmpConfig()
	StartMetricsWithConfig(
		nCfg,
		slog,
		TestName,
		TestVersion,
		TestNetworkID,
		*TestCoinbase,
	)
	markTest()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			if result != nil && strings.Contains(*result, "test.Gauge") {
				return
			}
		}
	}()
	wg.Wait()

	if !strings.Contains(*result, "test.Gauge") {
		t.Fatal("get test.Gauge failed")
	}
	if !strings.Contains(*result, "test.Counter") {
		t.Fatal("get test.Counter failed")
	}
	if !strings.Contains(*result, "test.Meter") {
		t.Fatal("get test.Meter failed")
	}
	if !strings.Contains(*result, "test.GaugeFloat64") {
		t.Fatal("get test.GaugeFloat64 failed")
	}
	if !strings.Contains(*result, "test.Histogram") {
		t.Fatal("get test.Histogram failed")
	}
	if !strings.Contains(*result, "test.Timer") {
		t.Fatal("get test.Timer failed")
	}
	if strings.Contains(*result, "not exsit") {
		t.Fatal("get a value of not exsit")
	}
}
