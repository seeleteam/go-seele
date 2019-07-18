package pbft

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/seeleteam/go-seele/log"
)

type State struct {
	ViewID         int64
	MsgLogs        *MsgLogs
	LastSequenceID int64
	CurrentStage   Stage
	log            *log.SeeleLog
}

type MsgLogs struct {
	ReqMsg      *RequestMsg
	PrepareMsgs map[string]*VoteMsg
	CommitMsgs  map[string]*VoteMsg
}

type Stage int

const (
	Idle Stage = iota
	PrePrepared
	Prepared
	Committed
)

// f: # of Byzantine faulty node
// f = n / 3
// f = 4, in this case
const f = 1

// CreateState create the state
func CreateState(viewID int64, lastSequenceID int64) *State {
	return &State{
		ViewID: viewID,
		MsgLogs: &MsgLogs{
			ReqMsg:      nil,
			PrepareMsgs: make(map[string]*VoteMsg),
			CommitMsgs:  make(map[string]*VoteMsg),
		},
		LastSequenceID: lastSequenceID,
		CurrentStage:   Idle,
		log:            log.GetLogger("pdft_engine"),
	}
}

// StartConsensus start the consensus
func (state *State) StartConsensus(request *RequestMsg) (*PrePrepareMsg, error) {
	sequenceID := time.Now().UnixNano() // index of the message

	// find the unique and the largest sequenceID
	if state.LastSequenceID != -1 { // -1: there is no last sequence ID
		for state.LastSequenceID >= sequenceID {
			sequenceID += 1
		}
	}
	// assign a new sequenceID to the request message
	request.SequenceID = sequenceID
	// save MsgLogs
	state.MsgLogs.ReqMsg = request

	digest, err := digest(request)
	if err != nil {
		state.log.Info("digest of request error:", err)
		return nil, err
	}

	// change the stage as pre-prepared
	state.CurrentStage = PrePrepared

	return &PrePrepareMsg{
		ViewID:     state.ViewID,
		SequenceID: sequenceID,
		Digest:     digest,
		RequestMsg: request,
	}, nil
}

func (state *State) PrePrepare(prePrepareMsg *PrePrepareMsg) (*VoteMsg, error) {
	// save RequestMsg to MsgLogs
	state.MsgLogs.ReqMsg = prePrepareMsg.RequestMsg

	// verify message
	if !state.verifyMsg(prePrepareMsg.ViewID, prePrepareMsg.SequenceID, prePrepareMsg.Digest) {
		return nil, error.New("fail to pre-prepare message")
	}

	// change the stage to pre-prepapred
	state.CurrentStage = PrePrepared
	return &VoteMsg{
		ViewID:     state.ViewID,
		SequenceID: prePrepareMsg.SequenceID,
		Digest:     prePrepareMsg.Digest,
		MsgType:    PrepareMsg,
	}, nil
}

func (state *State) Prepare(prepareMsg *VoteMsg) (*VoteMsg, error) {
	// verfify message
	if !state.verifyMsg(prepareMsg.ViewID, prepareMsg.SequenceID, prepareMsg.Digest) {
		state.log.Info("prepareMsg verification failed")
		return nil, errors.New("prepare message failed")
	}

	//add msg
	state.MsgLogs.PrepareMsgs[prepareMsg.NodeID] = prepareMsg
	state.log.Info("[Prepare-Vote]: %d\n", len(state.MsgLogs.PrepareMsgs))

	if state.isPrepared() {
		// change current stage to prepared
		state.CurrentStage = Prepared

		return &VoteMsg{
			ViewID:     state.ViewID,
			SequenceID: prepareMsg.SequenceID,
			Digest:     prepareMsg.Digest,
			MsgType:    CommitMsg,
		}, nil
	}
}

func (state *State) Commit(commitMsg *VoteMsg) (*ReplyMsg, *RequestMsg, error) {
	if !state.verifyMsg(commitMsg.ViewID, commitMsg.SequenceID, commitMsg.Digest) {
		return nil, nil, errors.New("commit message failed")
	}
	// Append msg
	state.MsgLogs.CommitMsgs[commitMsg.NodeID] = commitMsg
	state.log.Info("[Commit-Vote]: %d", len(state.MsgLogs.CommitMsgs))

	if state.isCommitted() {
		// This node executes the requested operation locally and gets the result.
		result := "Executed"
		state.CurrentStage = Committed
		return &ReplyMsg{
			ViewID:    state.ViewID,
			Timestamp: state.MsgLogs.ReqMsg.Timestamp,
			ClientID:  state.MsgLogs.ReqMsg.ClientID,
			Result:    result,
		}, state.MsgLogs.ReqMsg, nil
	}
	return nil, nil, nil
}
func (state *State) isCommitted() bool {
	if !state.isPrepared() {
		return false
	}
	if len(state.MsgLogs.CommitMsgs) < 2*f {
		return false
	}
	return true
}
func (state *State) isPrepared() bool {
	if state.MsgLogs.ReqMsg == nil {
		return false
	}
	if len(state.MsgLogs.PrepareMsgs) < 2*f {
		return false
	}
	return true
}

// verifyMsg verify the message
func (state *State) verifyMsg(viewID int64, sequenceID int64, digestGot string) bool {
	// Wrong view. That is, wrong configurations of peers to start the consensus.
	if state.ViewID != viewID {
		return false
	}

	// Check if the Primary sent fault sequence number. => Faulty primary.
	// TODO: adopt upper/lower bound check.
	if state.LastSequenceID != -1 {
		if state.LastSequenceID >= sequenceID {
			return false
		}
	}

	digest, err := digest(state.MsgLogs.ReqMsg)
	if err != nil {
		fmt.Println(err)
		return false
	}

	// Check digest.
	if digestGot != digest {
		return false
	}

	return true
}

// digest return the hash value of input object
func digest(object interface{}) (string, error) {
	msg, err := json.Marshal(object)
	if err != nil {
		return "", err
	}
	return Hash(msg), nil
}

// Hash hash the content
func Hash(content []byte) string {
	h := sha256.New()
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}
