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
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
)

const (
	TestName      = "seele node1"
	TestVersion   = "1.0"
	TestNetworkID = "seele"
)

var (
	TestCoinbase = crypto.MustGenerateShardAddress(1)
	slog         = log.GetLogger("seele")
	address      = "127.0.0.1:8086"
	result       = new(string)
	mux          sync.Mutex
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

	mux.Lock()
	*result = *result + string(s)
	mux.Unlock()
}

// influxdbSimulate simulate the influxdb server
func influxdbSimulate() {
	http.HandleFunc("/write", saveResult)
	err := http.ListenAndServe(address, nil)
	if err != nil {
		slog.Fatal("ListenAndServe: %s", err)
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
		defer mux.Unlock()

		for {
			mux.Lock()
			if result != nil && strings.Contains(*result, "test.Gauge") {
				return
			}
			mux.Unlock()
		}
	}()
	wg.Wait()

	resultCompare(t, "test.Gauge", "failed to get test.Gauge")
	resultCompare(t, "test.Counter", "failed to get test.Counter")
	resultCompare(t, "test.Meter", "failed to get test.Meter")
	resultCompare(t, "test.GaugeFloat64", "failed to get test.GaugeFloat64")
	resultCompare(t, "test.Histogram", "failed to get test.Histogram")
	resultCompare(t, "test.Timer", "failed to get test.Timer")
	resultCompareContains(t, "not exsit", "get a value of not exsit")
}

func resultCompare(t *testing.T, data string, errMsg string) {
	defer mux.Unlock()

	mux.Lock()
	if !strings.Contains(*result, data) {
		t.Fatal(errMsg)
	}
}

func resultCompareContains(t *testing.T, data string, errMsg string) {
	defer mux.Unlock()

	mux.Lock()
	if strings.Contains(*result, data) {
		t.Fatal(errMsg)
	}
}
