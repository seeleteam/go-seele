package discovery

import (
	"testing"
	"github.com/seeleteam/go-seele/common"
)

func Test_SaveM2File(t *testing.T) {
	var key common.Hash
	te := []byte{1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,1,2,3,4,5,6,7,8,9,0,2,2}
	copy(key[:],te[:])
	m := map[common.Hash]*Node{
		key:&Node{
			UDPPort:66,
			TCPPort:66,
		},
	}
	SaveM2File(m)
}