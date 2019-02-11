/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package lightclients

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/hashicorp/golang-lru"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/light"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/node"
)

var (
	errWrongShardDebt = errors.New("wrong debt with invalid shard")
	errNotMatchedTx   = errors.New("transaction mismatch with request debt")
	errNotFoundTx     = errors.New("not found debt's transaction")
)

// LightClientsManager manages light clients of other shards and provides services for debt validation.
type LightClientsManager struct {
	lightClients        []*light.ServiceClient
	lightClientsBackend []*light.LightBackend
	confirmedTxs        []*lru.Cache
	packedDebts         []*lru.Cache

	localShard uint
}

// NewLightClientManager create a new LightClientManager instance.
func NewLightClientManager(targetShard uint, context context.Context, config *node.Config, engine consensus.Engine) (*LightClientsManager, error) {
	clients := make([]*light.ServiceClient, common.ShardCount+1)
	backends := make([]*light.LightBackend, common.ShardCount+1)
	confirmedTxs := make([]*lru.Cache, common.ShardCount+1)
	packedDebts := make([]*lru.Cache, common.ShardCount+1)

	copyConf := config.Clone()
	var err error
	for i := 1; i <= common.ShardCount; i++ {
		if i == int(targetShard) {
			continue
		}

		shard := uint(i)
		copyConf.SeeleConfig.GenesisConfig.ShardNumber = shard

		dbFolder := filepath.Join("db", fmt.Sprintf("lightchainforshard_%d", i))
		clients[i], err = light.NewServiceClient(context, copyConf, log.GetLogger(fmt.Sprintf("lightclient_%d", i)), dbFolder, shard, engine)
		if err != nil {
			return nil, err
		}

		backends[i] = light.NewLightBackend(clients[i])

		// At most, shardCount * 8K (txs+dets) hash values cached.
		// In case of 8 shards, 64K hash values cached, consuming about 2M memory.
		confirmedTxs[i] = common.MustNewCache(4096)
		packedDebts[i] = common.MustNewCache(4096)
	}

	return &LightClientsManager{
		lightClients:        clients,
		lightClientsBackend: backends,
		confirmedTxs:        confirmedTxs,
		packedDebts:         packedDebts,
		localShard:          targetShard,
	}, nil
}

// ValidateDebt validate debt
// returns packed whether debt is packed
// returns confirmed whether debt is confirmed
// returns retErr error info
func (manager *LightClientsManager) ValidateDebt(debt *types.Debt) (packed bool, confirmed bool, retErr error) {
	fromShard := debt.Data.From.Shard()
	if fromShard == 0 || fromShard == manager.localShard {
		return false, false, errWrongShardDebt
	}

	// check cache first
	cache := manager.confirmedTxs[fromShard]
	if _, ok := cache.Get(debt.Data.TxHash); ok {
		return true, true, nil
	}

	// comment out for test only
	backend := manager.lightClientsBackend[fromShard]
	tx, index, err := backend.GetTransaction(backend.TxPoolBackend(), backend.ChainBackend().GetStore(), debt.Data.TxHash)
	if err != nil {
		return false, false, errors.NewStackedErrorf(err, "failed to get tx %v", debt.Data.TxHash)
	}

	if index == nil {
		return false, false, errNotFoundTx
	}

	checkDebt := types.NewDebtWithoutContext(tx)
	if checkDebt == nil || !checkDebt.Hash.Equal(debt.Hash) {
		return false, false, errNotMatchedTx
	}

	header := backend.ChainBackend().CurrentHeader()
	duration := header.Height - index.BlockHeight
	if duration < common.ConfirmedBlockNumber {
		return true, false, fmt.Errorf("invalid debt because not enough confirmed block number, wanted is %d, actual is %d", common.ConfirmedBlockNumber, duration)
	}

	// cache the confirmed tx
	cache.Add(debt.Data.TxHash, true)

	return true, true, nil
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

// IfDebtPacked indicates whether the specified debt is packed.
// returns packed whether debt is packed
// returns confirmed whether debt is confirmed
// returns retErr this error is return when debt is found invalid. which means we need remove this debt.
func (manager *LightClientsManager) IfDebtPacked(debt *types.Debt) (packed bool, confirmed bool, retErr error) {
	toShard := debt.Data.Account.Shard()
	if toShard == 0 || toShard == manager.localShard {
		return false, false, errWrongShardDebt
	}

	//check cache first
	cache := manager.packedDebts[toShard]
	if _, ok := cache.Get(debt.Hash); ok {
		return true, true, nil
	}

	backend := manager.lightClientsBackend[toShard]
	result, index, err := backend.GetDebt(debt.Hash)
	if err != nil {
		return false, false, errors.NewStackedErrorf(err, "failed to get debt %v", debt.Hash)
	}

	if index == nil {
		return false, false, nil
	}

	_, err = result.Validate(nil, false, toShard)
	if err != nil {
		return false, false, errors.NewStackedError(err, "failed to validate debt")
	}

	// only marked as packed when the debt is confirmed
	header := backend.ChainBackend().CurrentHeader()
	if header.Height-index.BlockHeight < common.ConfirmedBlockNumber {
		return true, false, nil
	}

	// cache the confirmed debt
	cache.Add(debt.Hash, true)

	return true, true, nil
}
