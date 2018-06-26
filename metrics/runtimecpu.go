package metrics

import (
	metrics "github.com/rcrowley/go-metrics"
	"os/exec"
	"fmt"
	"strconv"
)

var (
	metricsCputMeter    = metrics.NewRegisteredMeter("cpu.accout", nil)
)

func getCPU() (int64,error){
	c := "top -d 1 | grep node | awk -F '[ ]+' '{print $10}'"
	cmd := exec.Command("sh", "-c", c)
	out, err := cmd.Output()
	fmt.Printf("cup data: %s\n", string(out))
	if err != nil {
		fmt.Printf("get cpu cmd failed: %s", err.Error())
		return *new(int64), err
	}

	i, err := strconv.ParseInt(string(out), 10, 64)
	if err != nil{
		fmt.Printf("data type conversion failed: %s", err.Error())
		return *new(int64), err
	}
	return i, nil
}