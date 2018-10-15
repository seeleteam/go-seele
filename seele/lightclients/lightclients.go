/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package lightclients

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/consensus/factory"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/light"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/node"
)

var (
	errWrongShardDebt = errors.New("wrong debt with invalid shard")
	errNotMatchedTx   = errors.New("transaction mismatch with request debt")
)

// LightClientsManager manages light clients of other shards and provides services for debt validation.
type LightClientsManager struct {
	lightClients        []*light.ServiceClient
	lightClientsBackend []*light.LightBackend

	localShard uint
}

// NewLightClientManager create a new LightClientManager instance.
func NewLightClientManager(targetShard uint, context context.Context, config *node.Config) (*LightClientsManager, error) {
	clients := make([]*light.ServiceClient, common.ShardCount+1)
	backends := make([]*light.LightBackend, common.ShardCount+1)

	copyConf := config.Clone()
	var err error
	for i := 1; i <= common.ShardCount; i++ {
		if i == int(targetShard) {
			continue
		}

		shard := uint(i)
		copyConf.SeeleConfig.GenesisConfig.ShardNumber = shard

		dbFolder := filepath.Join("db", fmt.Sprintf("lightchainforshard_%d", i))
		var engine consensus.Engine
		engine, err = factory.GetConsensusEngine(copyConf.BasicConfig.MinerAlgorithm)
		if err != nil {
			return nil, err
		}

		clients[i], err = light.NewServiceClient(context, copyConf, log.GetLogger(fmt.Sprintf("lightclient_%d", i)), dbFolder, shard, engine)
		if err != nil {
			return nil, err
		}

		backends[i] = light.NewLightBackend(clients[i])
	}

	return &LightClientsManager{
		lightClients:        clients,
		lightClientsBackend: backends,
		localShard:          targetShard,
	}, nil
}

// ValidateDebt validate debt
func (manager *LightClientsManager) ValidateDebt(debt *types.Debt) (bool, error) {
	if debt.Data.Shard == 0 || debt.Data.Shard == manager.localShard {
		return false, errWrongShardDebt
	}

	backend := manager.lightClientsBackend[int(debt.Data.Shard)]
	tx, index, err := backend.GetTransaction(backend.TxPoolBackend(), backend.ChainBackend().GetStore(), debt.Data.TxHash)
	if err != nil {
		return false, err
	}

	if index == nil {
		return false, nil
	}

	header := backend.ChainBackend().CurrentHeader()
	duration := header.Height - index.BlockHeight
	if duration < common.ConfirmedBlockNumber {
		return false, fmt.Errorf("invalid debt because not enough confirmed block number, wanted is %d, actual is %d", common.ConfirmedBlockNumber, duration)
	}

	checkDebt := types.NewDebt(tx)
	if checkDebt == nil || !checkDebt.Hash.Equal(debt.Hash) {
		return false, errNotMatchedTx
	}

	return true, nil
}

// GetServices get node service
func (manager *LightClientsManager) GetServices() []node.Service {
	services := make([]node.Service, 0)
	for _, s := range manager.lightClients {
		if s != nil {
			services = append(services, s)
		}
	}

	return services
}
