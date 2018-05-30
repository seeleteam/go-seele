package discovery

import (
	"testing"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/stretchr/testify/assert"
)

func Test_SaveNodes(t *testing.T) {
	fileFullPath := common.GetNodeBackups()

	var key1 common.Hash
	var key2 common.Hash
	str := "12345678901234567890123456789022"
	te := []byte(str)
	copy(key1[:], te[:])
	copy(key2[1:], te[:])
	m := map[common.Hash]*Node{
		key1: &Node{
			UDPPort: 66,
			TCPPort: 66,
		},
		key2: &Node{
			UDPPort: 86,
			TCPPort: 86,
		},
	}

	db := Database{
		m: m,
	}
	go db.SaveNodes()
	time.Sleep(2 * time.Second)
	assert.Equal(t, common.FileOrFolderExists(fileFullPath), true)
}
