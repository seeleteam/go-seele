/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package core

import "github.com/seeleteam/go-seele/common"

func (c *core) handleFinalCommitted() error {
	logger := c.logger.New("state", c.state)
	logger.Trace("Received a final committed proposal")
	c.startNewRound(common.Big0)
	return nil
}
