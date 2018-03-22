/**
*  @file
*  @copyright defined in go-seele/LICENSE
*/

package node

import (
	"github.com/seeleteam/go-seele/p2p"
)

// Config holds Node options.
type Config struct {
	// The name of the node
	Name string 

	// The version of the node
	Version string 
	
	// The file system folder of the node use to store data
	DataDir string 

	// The configuration of p2p network
	P2P p2p.Config
}
