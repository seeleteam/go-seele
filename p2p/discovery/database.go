/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package discovery

import (
	"github.com/seeleteam/go-seele/common"
	"sync"
)

type database struct {
	m map[common.Hash]*Node // TODO use memory for temp, will use level db later

	mutex sync.Mutex
}

func NewDatabase() *database {
	return &database{
		m: make(map[common.Hash]*Node),
	}
}

func (db *database) add(id common.Hash, value *Node)  {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	db.m[id] = value
}

func (db *database) find(id common.Hash) *Node {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	return db.m[id]
}

func (db *database) delete(id common.Hash)  {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	delete(db.m, id)
}

func (db *database) getCopy() map[common.Hash]*Node {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	copyMap := make(map[common.Hash]*Node)
	for key, value := range db.m {
		copyMap[key] = value
	}

	return copyMap
}