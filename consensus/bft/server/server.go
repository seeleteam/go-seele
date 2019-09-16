package server

import (
	"crypto/ecdsa"

	BFT "github.com/seeleteam/go-seele/consensus/bft"
	"github.com/seeleteam/go-seele/database"
)

// NeServer new a server for bft backend.
func NewServer(config *BFT.BFTConfig, privateKey *ecdsa.PrivateKey, db database.Database) {

}
