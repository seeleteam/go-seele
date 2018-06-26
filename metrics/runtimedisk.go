package metrics

import (
	metrics "github.com/rcrowley/go-metrics"
	"github.com/shirou/gopsutil/disk"
	"fmt"
)

var (
	metricsDiskReadCountGauge  = metrics.GetOrRegisterGauge("disk.read.count", nil)
	metricsDiskReadBytesGauge  = metrics.GetOrRegisterGauge("disk.read.bytes", nil)
	metricsDiskWriteCountGauge  = metrics.GetOrRegisterGauge("disk.write.count", nil)
	metricsDiskWriteBytesGauge  = metrics.GetOrRegisterGauge("disk.write.bytes", nil)
)

func getDiskRate(name string) (disk.IOCountersStat,error){
	out,err := disk.IOCounters(name)
	if err != nil {
		fmt.Printf("get cpu cmd failed: %s", err.Error())
		return *new(disk.IOCountersStat), err
	}
	resul :=  disk.IOCountersStat{}
	for _,v := range out{
		resul.ReadBytes = resul.ReadBytes + v.ReadBytes
		resul.ReadCount = resul.ReadCount + v.ReadCount
		resul.WriteBytes = resul.WriteBytes + v.WriteBytes
		resul.WriteCount = resul.WriteCount + v.WriteCount
	}
	fmt.Printf("disk data: %v\n", resul)
	return resul,nil
}
