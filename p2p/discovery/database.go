/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"sync"

	"github.com/seeleteam/go-seele/common"
)

type Database struct {
	m map[common.Hash]*Node // TODO use memory for temp, will use level db later

	mutex sync.Mutex
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
	db.m[*sha] = value
}

func (db *Database) find(id common.Hash) *Node {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	return db.m[id]
}

func (db *Database) delete(id *common.Hash) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	delete(db.m, *id)
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
