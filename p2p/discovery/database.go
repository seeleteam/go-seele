/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
)

type NodeHook func(node *Node)

type Database struct {
	m              map[common.Hash]*Node
	log            *log.SeeleLog
	mutex          sync.Mutex
	addNodeHook    NodeHook
	deleteNodeHook NodeHook
}

// StartSaveNodes will save to a file and open a timer to backup the nodes info
func (db *Database) StartSaveNodes(nodeDir string, done chan bool) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			db.log.Debug("backups nodes...\n")
			go db.SaveNodes(nodeDir)
		case <-done:
			return
		}
	}
}

func (db *Database) SaveNodes(nodeDir string) {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	if db.m == nil {
		return
	}
	filePath := filepath.Join(common.GetDefaultDataFolder(), nodeDir)
	fileFullPath := filepath.Join(filePath, "nodes.txt")

	nodeStr := make([]string, len(db.m))
	i := 0
	for _, v := range db.m {
		nodeStr[i] = v.String()
		i++
	}

	nodeByte, err := json.MarshalIndent(nodeStr, "", "\t")
	if err != nil {
		db.log.Error("json marshal occur error, for:[%s]", err.Error())
		return
	}

	if !common.FileOrFolderExists(fileFullPath) {
		if err := os.MkdirAll(filePath, os.ModePerm); err != nil {
			db.log.Error("filePath:[%s] create folder failed, for:[%s]", filePath, err.Error())
			return
		}
	}

	if err = ioutil.WriteFile(fileFullPath, nodeByte, 0666); err != nil {
		db.log.Error("nodes info backup failed, for:[%s]", err.Error())
		return
	}
	db.log.Info("nodes:%s info backup success\n", nodeByte)
}

func NewDatabase(log *log.SeeleLog) *Database {
	return &Database{
		m:   make(map[common.Hash]*Node),
		log: log,
	}
}

func (db *Database) add(value *Node) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	sha := value.getSha()
	if _, ok := db.m[sha]; !ok && db.addNodeHook != nil {
		go db.addNodeHook(value)
	}

	db.m[sha] = value
}

func (db *Database) FindByNodeID(id common.Address) (*Node, bool) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	sha := crypto.HashBytes(id.Bytes())
	val, ok := db.m[sha]

	return val, ok
}

func (db *Database) delete(id common.Hash) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	if val, ok := db.m[id]; ok && db.deleteNodeHook != nil {
		go db.deleteNodeHook(val)
	}

	delete(db.m, id)
}

func (db *Database) getRandNodes(number int) []*Node {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	nodes := make([]*Node, 0)
	count := 0
	for _, value := range db.m {
		if count == number {
			break
		}

		nodes = append(nodes, value)
		count++
	}

	return nodes
}

func (db *Database) getRandNode() *Node {
	nodes := db.getRandNodes(1)
	if len(nodes) != 1 {
		return nil
	}

	return nodes[0]
}

func (db *Database) size() int {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	return len(db.m)
}

func (db *Database) GetCopy() map[common.Hash]*Node {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	copyMap := make(map[common.Hash]*Node)
	for key, value := range db.m {
		copyMap[key] = value
	}

	return copyMap
}

// SetHookForNewNode this hook will be called when find new Node
// Note it will run in a new go routine
func (db *Database) SetHookForNewNode(hook NodeHook) {
	db.addNodeHook = hook
}

// SetHookForDeleteNode this hook will be called when we lost a Node's connection
// Note it will run in a new go routine
func (db *Database) SetHookForDeleteNode(hook NodeHook) {
	db.deleteNodeHook = hook
}
