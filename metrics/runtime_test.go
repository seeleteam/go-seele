package metrics

import (
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/seeleteam/go-seele/common"
)

func Test_getCPURate(t *testing.T) {
	_, err := getCPURate(common.MetricsRefreshTime, false)
	assert.Equal(t, err, nil)
}
