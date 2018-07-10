package metrics

import (
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
)

func Test_getCPURate(t *testing.T) {
	_, err := getCPURate(100*time.Millisecond, false)
	assert.Equal(t, err, nil)
}
