package core

import "github.com/seeleteam/go-seele/common"

func (c *core) handleFinalCommit() error {
	c.log.Debug("received a final commit proposal")
	c.startNewRound(common.Big0)
	return nil
}
