/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package factory

import (
	"crypto/ecdsa"
	"fmt"
	"path/filepath"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/consensus/bft"
	"github.com/seeleteam/go-seele/consensus/bft/server"
	"github.com/seeleteam/go-seele/consensus/ethash"
	"github.com/seeleteam/go-seele/consensus/istanbul"
	"github.com/seeleteam/go-seele/consensus/istanbul/backend"
	"github.com/seeleteam/go-seele/consensus/pow"
	"github.com/seeleteam/go-seele/consensus/spow"
	"github.com/seeleteam/go-seele/database/leveldb"
)

// GetConsensusEngine get consensus engine according to miner algorithm name
// WARNING: engine may be a heavy instance. we should have as less as possible in our process.
func GetConsensusEngine(minerAlgorithm string, folder string) (consensus.Engine, error) {
	var minerEngine consensus.Engine
	if minerAlgorithm == common.EthashAlgorithm {
		minerEngine = ethash.New(ethash.GetDefaultConfig(), nil, false)
	} else if minerAlgorithm == common.Sha256Algorithm {
		minerEngine = pow.NewEngine(1)
	} else if minerAlgorithm == common.SpowAlgorithm {
		minerEngine = spow.NewSpowEngine(1, folder)
	} else {
		return nil, fmt.Errorf("unknown miner algorithm")
	}

	return minerEngine, nil
}

func GetBFTEngine(privateKey *ecdsa.PrivateKey, folder string) (consensus.Engine, error) {
	path := filepath.Join(folder, common.BFTDataFolder)
	db, err := leveldb.NewLevelDB(path)
	if err != nil {
		return nil, errors.NewStackedError(err, "create bft folder failed")
	}

	return backend.New(istanbul.DefaultConfig, privateKey, db), nil
}

func MustGetConsensusEngine(minerAlgorithm string) consensus.Engine {
	engine, err := GetConsensusEngine(minerAlgorithm, "temp")
	if err != nil {
		panic(err)
	}

	return engine
}

// subchain bft engine entrance
func GetBFTSubchainEngine(privateKey *ecdsa.PrivateKey, folder string) (consensus.Engine, error) {
	path := filepath.Join(folder, common.BFTDataFolder)
	db, err := leveldb.NewLevelDB(path)
	if err != nil {
		return nil, errors.NewStackedError(err, "create bft folder failed")
	}

	return server.NewServer(bft.DefaultConfig, privateKey, db), nil
}
