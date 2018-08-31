/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/log"
)

type odrBackend struct {
	log *log.SeeleLog
}

func newOdrBackend(log *log.SeeleLog) *odrBackend {

	return &odrBackend{
		log: log,
	}
}

// getBlock retrieves block body from network.
func (o *odrBackend) getBlock(hash common.Hash, no uint64) (*types.Block, error) {
	return nil, nil
}

func (o *odrBackend) close() {

}
