package metrics

import (
	"runtime"
	"testing"

	"github.com/magiconair/properties/assert"
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
