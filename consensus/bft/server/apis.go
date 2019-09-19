package server

import "github.com/seeleteam/go-seele/consensus"

type API struct {
	chain consensus.ChainReader
	bft   *server
}

// define all apis here
