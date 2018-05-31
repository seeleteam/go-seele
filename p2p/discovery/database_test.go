package discovery

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log"
	"github.com/stretchr/testify/assert"
)

func Test_SaveNodes(t *testing.T) {
	fileFullPath := filepath.Join(common.GetDefaultDataFolder(), "node1", "nodes.txt")
	str1 := "12345678901234567890123456789022"
	str2 := "12345678901234567890123456789026"
	key1 := common.StringToHash(str1)
	key2 := common.StringToHash(str2)

	log := log.GetLogger("discovery", common.LogConfig.PrintLog)

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

	db := NewDatabase(log)
	db.m = m
	go db.SaveNodes("node1")
	time.Sleep(2 * time.Second)
	assert.Equal(t, common.FileOrFolderExists(fileFullPath), true)
}
