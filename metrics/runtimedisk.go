package metrics

import (
	metrics "github.com/rcrowley/go-metrics"
	"github.com/shirou/gopsutil/disk"
)

var (
	metricsDiskFreeCountGauge  = metrics.GetOrRegisterGauge("disk.free.count", nil)
	metricsDiskReadCountGauge  = metrics.GetOrRegisterGauge("disk.read.count", nil)
	metricsDiskWriteCountGauge = metrics.GetOrRegisterGauge("disk.write.count", nil)
	metricsDiskReadBytesGauge  = metrics.GetOrRegisterGauge("disk.read.bytes", nil)
	metricsDiskWriteBytesGauge = metrics.GetOrRegisterGauge("disk.write.bytes", nil)
	metricsDiskIoTimeGauge     = metrics.GetOrRegisterGauge("disk.io.time", nil)
)

func GetDiskInfo(name string) *disk.IOCountersStat {
	searchResult, err := disk.IOCounters(name)
	if err != nil {
		return nil
	}

	result := disk.IOCountersStat{}
	for _, v := range searchResult {
		result.WriteBytes = result.WriteBytes + v.WriteBytes
		result.ReadBytes = result.ReadBytes + v.ReadBytes
		result.WriteCount = result.WriteCount + v.WriteCount
		result.ReadCount = result.ReadCount + v.ReadCount
		result.IoTime = result.IoTime + v.IoTime
	}
	return &result
}
