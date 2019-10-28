package server

import (
	"errors"
	"fmt"

	lru "github.com/hashicorp/golang-lru"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/consensus/bft"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/p2p"
)

/*this file will implement all methods at consensus/consensus.go Handler interface*/

const (
	bftMsg = 0x11
)

// define your errors here
var (
	errDecodeFailed = errors.New("fail to decode bft message")
)

func (s *server) Protocal() consensus.Protocol {
	return consensus.Protocol{
		Name:     "bft",
		Versions: []uint{64}, //?
		Lengths:  []uint64{18},
	}
}

// HandleMsg implements consensus.Handler.HandleMsg
func (s *server) HandleMsg(addr common.Address, message interface{}) (bool, error) {
	s.coreMu.Lock()
	defer s.coreMu.Unlock()

	msg, ok := message.(p2p.Message)
	if !ok {
		return false, errDecodeFailed
	}

	// make msg type is right
	if msg.Code == bftMsg {
		// if core is not started
		if !s.coreStarted {
			return true, bft.ErrEngineStopped
		}
		var data []byte
		if err := common.Deserialize(msg.Payload, &data); err != nil {
			return true, errDecodeFailed
		}
		hash := crypto.HashBytes(data)

		// handle peer's message
		var m *lru.ARCCache
		ms, ok := s.recentMessages.Get(hash)

		if ok {
			m, _ = ms.(*lru.ARCCache)
		} else {
			m, _ = lru.NewARC(inmemoryMessages)
			s.recentMessages.Add(addr, m)
		}
		m.Add(hash, true)

		// handle self know message
		if _, ok := s.knownMessages.Get(hash); ok {
			return true, nil
		}
		s.knownMessages.Add(hash, true)

		go s.bftEventMux.Post(bft.MessageEvent{ // post all
			Payload: data,
		})
		fmt.Println("Post in HandleMsg")

		return true, nil
	}

	return false, nil
}

func (s *server) SetBroadcaster(broadcaster consensus.Broadcaster) {
	s.broadcaster = broadcaster
}

func (s *server) HandleNewChainHead() error {
	s.coreMu.RLock()
	defer s.coreMu.RUnlock()

	if !s.coreStarted {
		return bft.ErrEngineStopped
	}

	go s.bftEventMux.Post(bft.FinalCommittedEvent{})
	return nil
}
