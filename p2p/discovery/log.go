/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package discovery

import (
	"github.com/seeleteam/go-seele/common"
	slog "github.com/seeleteam/go-seele/log"
)

var log *slog.SeeleLog

func init() {
	log = slog.GetLogger("discovery", common.PrintLog)
}
