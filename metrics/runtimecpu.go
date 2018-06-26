package metrics

import (
	"fmt"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	"github.com/shirou/gopsutil/cpu"
)

var (
	metricsCputGauge = metrics.GetOrRegisterGauge("cpu.accout", nil)
)

func getCPURate(interval time.Duration, percpu bool) (int64, error) {
	out, err := cpu.Percent(interval, percpu)
	if err != nil {
		fmt.Printf("get cpu cmd failed: %s", err.Error())
		return *new(int64), err
	}
	var resul float64
	for i := 1; i <= len(out); i++ {
		resul = resul + out[i-1]
	}
	fmt.Printf("cup data: %v\n", resul)
	return int64(resul), nil
}
