package metrics

import (
	"testing"
	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
)

func Test_getCPURate(t *testing.T) {
	_,err := getCPURate(refresh, false)
	assert.Equal(t, err, nil)
}

func Test_getDiskRate(t *testing.T) {
	_,err := getDiskRate(common.GetTempFolder())
	assert.Equal(t, err, nil)
}
