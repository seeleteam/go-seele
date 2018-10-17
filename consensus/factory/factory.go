/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package factory

import (
	"fmt"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/consensus/ethash"
	"github.com/seeleteam/go-seele/consensus/pow"
)

// GetConsensusEngine get consensus engine according to miner algorithm name
// WARNING: engine may be a heavy instance. we should have as less as possible in our process.
func GetConsensusEngine(minerAlgorithm string) (consensus.Engine, error) {
	var minerEngine consensus.Engine
	if minerAlgorithm == common.EthashAlgorithm {
		minerEngine = ethash.New(ethash.GetDefaultConfig(), nil, false)
	} else if minerAlgorithm == common.Sha256Algorithm {
		minerEngine = pow.NewEngine(1)
	} else {
		return nil, fmt.Errorf("unknown miner algorithm")
	}

	return minerEngine, nil
}

func MustGetConsensusEngine(minerAlgorithm string) consensus.Engine {
	engine, err := GetConsensusEngine(minerAlgorithm)
	if err != nil {
		panic(err)
	}

	return engine
}
