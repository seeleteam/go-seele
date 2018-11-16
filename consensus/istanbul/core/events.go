/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package core

import (
	"github.com/seeleteam/go-seele/consensus/istanbul"
)

type backlogEvent struct {
	src istanbul.Validator
	msg *message
}

type timeoutEvent struct{}
