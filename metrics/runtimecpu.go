package metrics

import (
	metrics "github.com/rcrowley/go-metrics"
	"os/exec"
	"fmt"
	"encoding/binary")

var (
	metricsCputMeter    = metrics.NewRegisteredMeter("cpu.accout", nil)
)

func getCPU() int64{
	c := "top -d 1 | grep node | awk -F '[ ]+' '{print $10}'"
	cmd := exec.Command("sh", "-c", c)
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("cpu: %v", out)
		return *new(int64)
	}

	return int64(binary.BigEndian.Uint64(out))
}