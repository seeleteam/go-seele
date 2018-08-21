/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"strings"
	"testing"

	"github.com/magiconair/properties/assert"
)

func Test_cap(t *testing.T) {
	protocol := newProtocol()
	cap := protocol.cap()

	assert.Equal(t, cap.Name, protocol.Name)
	assert.Equal(t, cap.Version, protocol.Version)
}

func Test_String(t *testing.T) {
	protocol := newProtocol()
	cap := protocol.cap()
	str := cap.String()

	assert.Equal(t, strings.Compare(str, "udp/1"), 0)
}

func newProtocol() *Protocol {
	return &Protocol{
		Name:    "udp",
		Version: 1,
		Length:  1048,
	}
}
