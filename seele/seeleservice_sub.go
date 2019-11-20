/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"context"
	"fmt"

	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/miner"
	"github.com/seeleteam/go-seele/node"
)

// NewSeeleServiceSubchain create SeeleService for subchain
func NewSeeleServiceSubchain(ctx context.Context, conf *node.Config, log *log.SeeleLog, engine consensus.Engine) (sb *SeeleService, err error) {
	sb = &SeeleService{
		log:        log,
		networkID:  conf.P2PConfig.NetworkID,
		netVersion: conf.BasicConfig.Version,
	}

	serviceContext := ctx.Value("ServiceContext").(ServiceContext)
	fmt.Printf("start to initiate service with conf %+v", conf)

	// Initialize blockchain DB.
	if err = sb.initBlockchainDB(&serviceContext); err != nil {
		return nil, err
	}

	leveldb.StartMetrics(sb.chainDB, "chaindb", log)

	// Initialize account state info DB.
	if err = sb.initAccountStateDB(&serviceContext); err != nil {
		return nil, err
	}

	sb.miner = miner.NewMinerSubchain(conf.SeeleConfig.Coinbase, sb, engine)

	// initialize and validate genesis
	fmt.Printf("[subchain] newSeeleService engine %+v", engine)
	if err = sb.initGenesisAndChain(&serviceContext, conf, -1); err != nil {
		return nil, err
	}

	if err = sb.initPool(conf); err != nil {
		return nil, err
	}

	if sb.seeleProtocol, err = NewSeeleProtocol(sb, log); err != nil {
		sb.Stop()
		log.Error("failed to create seeleProtocol in NewSeeleService, %s", err)
		return nil, err
	}
	return sb, nil
}
