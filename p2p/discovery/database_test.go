package discovery

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"

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
			UDPPort: 66,
			TCPPort: 66,
		},
	}

	db := NewDatabase(log)
	db.m = m
	db.SaveNodes("node1")
	assert.Equal(t, common.FileOrFolderExists(fileFullPath), true)
	data, err := ioutil.ReadFile(fileFullPath)
	assert.Equal(t, err, nil)
	cnode := make([]string, 2)
	err = json.Unmarshal(data, &cnode)
	assert.Equal(t, err, nil)
	assert.Equal(t, cnode[0], "snode://00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000@:66[0]")
	assert.Equal(t, len(cnode), 2)
}
