/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package discovery

import (
	"github.com/seeleteam/go-seele/common"
)

type database struct {
	m map[common.Hash]*Node // TODO use memory for temp, will use level db later
}

func NewDatabase() *database {
	return &database{
		m: make(map[common.Hash]*Node),
	}
}

func (db *database) add(id common.Hash, value *Node)  {
	db.m[id] = value
}

func (db *database) find(id common.Hash) *Node {
	return db.m[id]
}

func (db *database) delete(id common.Hash)  {
	delete(db.m, id)
}