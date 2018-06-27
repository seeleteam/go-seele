package metrics

import (
	metrics "github.com/rcrowley/go-metrics"
	"github.com/shirou/gopsutil/disk"
)

var (
	metricsDiskReadCountGauge  = metrics.GetOrRegisterGauge("disk.read.count", nil)
	metricsDiskReadBytesGauge  = metrics.GetOrRegisterGauge("disk.read.bytes", nil)
	metricsDiskWriteCountGauge = metrics.GetOrRegisterGauge("disk.write.count", nil)
	metricsDiskWriteBytesGauge = metrics.GetOrRegisterGauge("disk.write.bytes", nil)
)

func getDiskRate(name string) (disk.IOCountersStat, error) {
	out, err := disk.IOCounters(name)
	if err != nil {
		return *new(disk.IOCountersStat), err
	}

	result := disk.IOCountersStat{}
	for _, v := range out {
		result.ReadBytes = result.ReadBytes + v.ReadBytes
		result.ReadCount = result.ReadCount + v.ReadCount
		result.WriteBytes = result.WriteBytes + v.WriteBytes
		result.WriteCount = result.WriteCount + v.WriteCount
	}
	return result, nil
}
