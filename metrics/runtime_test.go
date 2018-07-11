package metrics

import (
	"runtime"
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
)

func Test_getCPURate(t *testing.T) {
	_, err := getCPURate(100*time.Millisecond, false)
	assert.Equal(t, err, nil)
}

func Test_getProcessCPURate(t *testing.T) {
	_, err := getProcessCPURate(100 * time.Millisecond)
	assert.Equal(t, err, nil)
}

func Test_GetDiskInfo(t *testing.T) {

	if result := GetDiskInfo(); result == nil {
		if runtime.GOOS == "linux" {
			t.Fatal("get disk info of linux machine failed")
		}
	} else {
		if runtime.GOOS != "linux" {
			t.Fatal("get disk info of windows machine failed")
		}
	}
}
