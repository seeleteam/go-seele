package network

import (
	"fmt"

	"github.com/seeleteam/go-seele/consensus/pbft/consensus"
)

func LogMsg(msg interface{}) {
	switch msg.(type) {
	case *consensus.RequestMsg:
		reqMsg := msg.(*consensus.RequestMsg)
		fmt.Printf("[REQUEST] ClientID: %s, Timestamp: %d, Operation: %s\n", reqMsg.ClientID, reqMsg.Timestamp, reqMsg.Operation)
	case *consensus.PrePrepareMsg:
		prePrepareMsg := msg.(*consensus.PrePrepareMsg)
		fmt.Printf("[PREPREPARE] ClientID: %s, Operation: %s, SequenceID: %d\n", prePrepareMsg.RequestMsg.ClientID, prePrepareMsg.RequestMsg.Operation, prePrepareMsg.SequenceID)
	case *consensus.VoteMsg:
		voteMsg := msg.(*consensus.VoteMsg)
		if voteMsg.MsgType == consensus.PrepareMsg {
			fmt.Printf("[PREPARE] NodeID: %s\n", voteMsg.NodeID)
		} else if voteMsg.MsgType == consensus.CommitMsg {
			fmt.Printf("[COMMIT] NodeID: %s\n", voteMsg.NodeID)
		}
	}
}

func LogStage(stage string, isDone bool) {
	if isDone {
		fmt.Printf("[STAGE-DONE] %s\n", stage)
	} else {
		fmt.Printf("[STAGE-BEGIN] %s\n", stage)
	}
}
