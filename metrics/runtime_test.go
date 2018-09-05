package metrics

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	metric "github.com/rcrowley/go-metrics"
)

func Test_getCPURate(t *testing.T) {
	_, err := getCPURate(0, false)
	assert.Equal(t, err, nil)
}

func Test_getProcessCPURate(t *testing.T) {
	_, err := getProcessCPURate(0)
	assert.Equal(t, err, nil)
}

func Test_GetDiskInfo(t *testing.T) {
	if result := GetDiskInfo(); result == nil {
		if runtime.GOOS == "linux" {
			t.Fatal("get the linux machine disk info failed")
		}
	} else {
		if runtime.GOOS != "linux" {
			t.Fatal("get the non linux machine disk info failed")
		}
	}
}

func Test_doMark(t *testing.T) {
	registry := metric.DefaultRegistry
	memStats := new(runtime.MemStats)
	var lastPauseNs uint64
	doMark(memStats, lastPauseNs)
	if registry.Get("runtime.memory.alloc") == nil {
		t.Fatal("get runtime.memory.alloc failed")
	}
	if registry.Get("runtime.memory.pauses") == nil {
		t.Fatal("get runtime.memory.pauses failed")
	}
	if registry.Get("cpu.os") == nil {
		t.Fatal("get cpu.os failed")
	}
	if registry.Get("cpu.seele") == nil {
		t.Fatal("get cpu.seele failed")
	}
	if registry.Get("disk.free.count") == nil {
		t.Fatal("get disk.free.count failed")
	}
	if registry.Get("disk.read.count") == nil {
		t.Fatal("get disk.read.count failed")
	}
	if registry.Get("disk.write.count") == nil {
		t.Fatal("get disk.write.count failed")
	}
	if registry.Get("disk.read.bytes") == nil {
		t.Fatal("get disk.read.bytes failed")
	}
	if registry.Get("disk.write.bytes") == nil {
		t.Fatal("get disk.write.bytes failed")
	}
	if registry.Get("not exsit") != nil {
		t.Fatal("get a value of not exsit")
	}
}
