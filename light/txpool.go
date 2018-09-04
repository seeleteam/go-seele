/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"github.com/seeleteam/go-seele/log"
)

type LightPool struct {
	odrBackend *odrBackend
	log        *log.SeeleLog
}

func newLightPool(chain BlockChain, odrBackend *odrBackend) (*LightPool, error) {
	pool := &LightPool{
		odrBackend: odrBackend,
	}

	return pool, nil
}
