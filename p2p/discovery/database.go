/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"fmt"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
)

type NodeHook func(node *Node)

type Database struct {
	m map[common.Hash]*Node // TODO use memory for temp, will use level db later

	mutex          sync.Mutex
	addNodeHook    NodeHook
	deleteNodeHook NodeHook
}

var dblog = log.GetLogger("discovery", common.LogConfig.PrintLog)

// SaveNodes will save to a file and open a timer to backups the nodes info
func (db *Database) SaveNodes() {
	saveNodes2File(db.m)
	startNewTicker(db.m)
}

func startNewTicker(m map[common.Hash]*Node) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			dblog.Debug("backups nodes\n")
			go saveNodes2File(m)
		}
	}
}

func saveNodes2File(m map[common.Hash]*Node) {
	if m == nil {
		return
	}
	filePath := common.GetDefaultDataFolder()
	fileFullPath := common.GetNodeBackupPath()

	nodeStr := make([]string, len(m))
	i := 0
	for k, v := range m {
		if i != len(m)-1 {
			nodeStr[i] = fmt.Sprintln(k.String() + "--" + v.String() + ",")
		} else {
			nodeStr[i] = fmt.Sprintln(k.String() + "--" + v.String())
		}
		i++
	}

	byteContent := fmt.Sprintln("[\n" + strings.Join(nodeStr, "") + "]")
	nodeByte := []byte(byteContent)
	if !common.FileOrFolderExists(fileFullPath) {
		err := os.MkdirAll(filePath, os.ModePerm)
		if err != nil {
			dblog.Error("filePath:[%s] create folder failed, for:[%s]", filePath, err.Error())
			return
		}
	}

	err := ioutil.WriteFile(fileFullPath, nodeByte, 0666)
	if err != nil {
		dblog.Error("nodes info backup failed, for:[%s]", err.Error())
		return
	}
	dblog.Debug("data:%s write to file\n", nodeByte)
}

func NewDatabase() *Database {
	return &Database{
		m: make(map[common.Hash]*Node),
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
