package metrics

import (
	metrics "github.com/rcrowley/go-metrics"
	"fmt"
	"strconv"
	"github.com/shirou/gopsutil/cpu"
	"time"
)

var (
	metricsCputMeter    = metrics.NewRegisteredMeter("cpu.accout", nil)
)

//func getCPU() (int64,error){
//	c := "top -d 1 | grep node | awk -F '[ ]+' '{print $9}'"
//	cmd := exec.Command("sh", "-c", c)
//	out, err := cmd.Output()
//	fmt.Printf("cup data: %s\n", string(out))
//	if err != nil {
//		fmt.Printf("get cpu cmd failed: %s", err.Error())
//		return *new(int64), err
//	}
//
//	i, err := strconv.ParseInt(string(out), 10, 64)
//	if err != nil{
//		fmt.Printf("data type conversion failed: %s", err.Error())
//		return *new(int64), err
//	}
//	return i, nil
//}

func getCPU(interval time.Duration, percpu bool) (int64,error){
	out,err := cpu.Percent(interval, percpu)
	var resul float64
	for i := 1; i <= len(out); i++{
		resul = resul + out[i - 1]
	}
	resul = resul / float64(len(out))
	result := strconv.FormatFloat(resul, 'E', -1, 32)
	fmt.Printf("cup data: %s\n", result)
	if err != nil {
		fmt.Printf("get cpu cmd failed: %s", err.Error())
		return *new(int64), err
	}

	i, err := strconv.ParseInt(result, 10, 64)
	if err != nil{
		fmt.Printf("data type conversion failed: %s", err.Error())
		return *new(int64), err
	}
	return i, nil
}
