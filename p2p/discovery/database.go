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
	mutex          sync.RWMutex
	addNodeHook    NodeHook
	deleteNodeHook NodeHook
}

const (
	// NodesBackupInterval is the nodes info of backup interval time
	NodesBackupInterval = time.Minute * 20

	// NodesBackupFileName is the nodes info of backup file name
	NodesBackupFileName = "nodes.json"
)

// StartSaveNodes will save to a file and open a timer to backup the nodes info
func (db *Database) StartSaveNodes(nodeDir string, done chan struct{}) {
	ticker := time.NewTicker(NodesBackupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			go db.SaveNodes(nodeDir)
		case <-done:
			return
		}
	}
}

func (db *Database) SaveNodes(nodeDir string) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()
	if db.m == nil {
		return
	}
	fileFullPath := filepath.Join(nodeDir, NodesBackupFileName)

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
		if err := os.MkdirAll(nodeDir, os.ModePerm); err != nil {
			db.log.Error("filePath:[%s], failed to create folder, for:[%s]", nodeDir, err.Error())
			return
		}
	}

	db.log.Info("backups nodes. node length %d", len(db.m))
	if err = ioutil.WriteFile(fileFullPath, nodeByte, 0666); err != nil {
		db.log.Error("nodes info backup failed, for:[%s]", err.Error())
		return
	}

	db.log.Debug("nodes:%s info backup success\n", string(nodeByte))
}

func NewDatabase(log *log.SeeleLog) *Database {
	return &Database{
		m:   make(map[common.Hash]*Node),
		log: log,
	}
}

func (db *Database) add(value *Node, notify bool) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	sha := value.getSha()
	if notify && db.addNodeHook != nil {
		go db.addNodeHook(value)
	}

	db.m[sha] = value
}

func (db *Database) FindByNodeID(id common.Address) (*Node, bool) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

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
	db.mutex.RLock()
	defer db.mutex.RUnlock()

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
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	return len(db.m)
}

func (db *Database) GetCopy() map[common.Hash]*Node {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

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
