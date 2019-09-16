package core

import (
	"sync"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus/bft"
)

type messageSet struct {
	view       *bft.View
	valSet     bft.VerifierSet
	messagesMu *sync.Mutex
	messages   map[common.Address]*message
}

func (ms *messageSet) Size() int {
	ms.messagesMu.Lock()
	defer ms.messagesMu.Unlock()
	return len(ms.messages)
}
