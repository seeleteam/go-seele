package metrics

import (
	metrics "github.com/rcrowley/go-metrics"
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"time"
)

var (
	metricsCputGauge    = metrics.GetOrRegisterGauge("cpu.accout", nil)
)

func getCPURate(interval time.Duration, percpu bool) (int64,error){
	out,err := cpu.Percent(interval, percpu)
	if err != nil {
		fmt.Printf("get cpu cmd failed: %s", err.Error())
		return *new(int64), err
	}
	var resul float64
	for i := 1; i <= len(out); i++{
		resul = resul + out[i - 1]
	}
	resul = resul / float64(len(out))
	fmt.Printf("cup data: %v\n", resul)
	return int64(resul),nil
}
