/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
)

type NodeHook func(node *Node)

type Database struct {
	m map[common.Hash]*Node // TODO use memory for temp, will use level db later

	mutex          sync.Mutex
	addNodeHook    NodeHook
	deleteNodeHook NodeHook
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
