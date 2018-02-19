/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package contract

import (
	slog "github.com/seeleteam/go-seele/log"
)

var log *slog.SeeleLog

func init()  {
	log = slog.GetLogger("contract", true)
}