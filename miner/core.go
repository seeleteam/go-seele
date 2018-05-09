/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package miner

import (
	"github.com/seeleteam/go-seele/log"
)

// logAbort logs the info that nonce finding is aborted
func logAbort(log *log.SeeleLog) {
	log.Info("nonce finding aborted")
}
