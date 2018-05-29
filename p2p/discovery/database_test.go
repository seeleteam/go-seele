package discovery

import (
	"testing"
	"github.com/seeleteam/go-seele/common"
)

func Test_SaveM2File(t *testing.T) {
	var key common.Hash
	str := "12345678901234567890123456789022"
	te := []byte(str)
	copy(key[:],te[:])
	m := map[common.Hash]*Node{
		key:&Node{
			UDPPort:66,
			TCPPort:66,
		},
	}
	SaveM2File(m)
}