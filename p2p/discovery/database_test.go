package discovery

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log"
	"github.com/stretchr/testify/assert"
)

func testNewDatabase() *Database {
	str1 := "12345678901234567890123456789022"
	str2 := "12345678901234567890123456789026"
	key1 := common.StringToHash(str1)
	key2 := common.StringToHash(str2)

	node1, err := NewNodeFromIP("127.0.0.1:6666")
	if err != nil {
		panic(err)
	}

	log := log.GetLogger("discovery")
	db := NewDatabase(log)

	m := map[common.Hash]*Node{
		key1: node1,
		key2: node1,
	}
	db.m = m

	return db
}

func Test_SaveNodes(t *testing.T) {
	fileFullPath := filepath.Join(common.GetTempFolder(), "nodes.json")
	db := testNewDatabase()
	db.SaveNodes(common.GetTempFolder())
	defer os.Remove(fileFullPath)

	assert.Equal(t, common.FileOrFolderExists(fileFullPath), true)
	data, err := ioutil.ReadFile(fileFullPath)
	assert.Equal(t, err, nil)
	cnode := make([]string, 2)
	err = json.Unmarshal(data, &cnode)
	assert.Equal(t, err, nil)
	assert.Equal(t, cnode[0], "snode://0000000000000000000000000000000000000000@127.0.0.1:6666[0]")
	assert.Equal(t, len(cnode), 2)
}

func Test_Database_GetRandNodes(t *testing.T) {
	db := testNewDatabase()

	nodes := db.getRandNodes(0)
	assert.Equal(t, len(nodes), 0)

	nodes = db.getRandNodes(1)
	assert.Equal(t, len(nodes), 1)

	nodes = db.getRandNodes(2)
	assert.Equal(t, len(nodes), 2)

	// only 2 nodes in db
	nodes = db.getRandNodes(3)
	assert.Equal(t, len(nodes), 2)

	nodes = db.getRandNodes(10)
	assert.Equal(t, len(nodes), 2)
}

func Test_Database_GetRandNode(t *testing.T) {
	// 2 nodes in this db
	db := testNewDatabase()
	node := db.getRandNode()
	assert.Equal(t, node != nil, true)

	// 0 nodes in this db
	log := log.GetLogger("discovery")
	db = NewDatabase(log)
	node = db.getRandNode()
	assert.Equal(t, node == nil, true)
}

func Test_Database_GetCopy(t *testing.T) {
	// 2 nodes in this db
	db := testNewDatabase()
	nodeMap := db.GetCopy()
	assert.Equal(t, nodeMap, db.m)

	// 0 nodes in this db
	log := log.GetLogger("discovery")
	db = NewDatabase(log)
	nodeMap = db.GetCopy()
	assert.Equal(t, len(nodeMap), 0)
}
