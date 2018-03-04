/**
*  @file
*  @copyright defined in go-seele/LICENSE
*/

package node

import (
	"github.com/seeleteam/go-seele/p2p"
)

// Config represents a small collection of configuration values to fine tune the
// P2P network layer of a protocol stack. These values can be further extended by
// all registered services.
type Config struct {
	// Name sets the instance name of the node.
	Name string `toml:"-"`

	// UserIdent
	UserIdent string `toml:",omitempty"`

	// Version
	Version string `toml:"-"`

	// DataDir is the file system folder the node 
	DataDir string

	// Configuration of peer-to-peer networking.
	P2P p2p.Config

	// KeyStoreDir is the file system folder that contains private keys
	KeyStoreDir string `toml:",omitempty"`
}


