package main

import (
	"os"

	"github.com/seeleteam/go-seele/consensus/pbft/network"
)

func main() {
	nodeID := os.Args[1]
	server := network.NewServer(nodeID)

	server.Start()
}

// // NewPBFTEngine start pbft consensus algorithm
// func NewPBFTEngine(nodeID string) {
// 	// nodeID := os.Args[1]
// 	server := network.NewServer(nodeID)

// 	server.Start()
// }
