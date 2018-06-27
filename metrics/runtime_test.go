package metrics

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
	"github.com/shirou/gopsutil/disk"
)

func Test_getCPURate(t *testing.T) {
	_, err := getCPURate(common.RefreshTime, false)
	assert.Equal(t, err, nil)
}
