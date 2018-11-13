/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package lightclients

import (
	"context"
	"fmt"
	"path/filepath"

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

	localShard uint
}

// NewLightClientManager create a new LightClientManager instance.
func NewLightClientManager(targetShard uint, context context.Context, config *node.Config, engine consensus.Engine) (*LightClientsManager, error) {
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
// returns packed whether debt is packed
// returns confirmed whether debt is confirmed
// returns retErr error info
func (manager *LightClientsManager) ValidateDebt(debt *types.Debt) (packed bool, confirmed bool, retErr error) {
	fromShard := debt.Data.From.Shard()
	if fromShard == 0 || fromShard == manager.localShard {
		retErr = errWrongShardDebt
		return
	}

	backend := manager.lightClientsBackend[fromShard]
	tx, index, err := backend.GetTransaction(backend.TxPoolBackend(), backend.ChainBackend().GetStore(), debt.Data.TxHash)
	if err != nil {
		retErr = errors.NewStackedError(err, "got error when get transaction.")
		return
	}

	if index == nil {
		retErr = errNotFoundTx
		return
	}

	checkDebt := types.NewDebtWithoutContext(tx)
	if checkDebt == nil || !checkDebt.Hash.Equal(debt.Hash) {
		retErr = errNotMatchedTx
		return
	}

	packed = true

	header := backend.ChainBackend().CurrentHeader()
	duration := header.Height - index.BlockHeight
	if duration >= common.ConfirmedBlockNumber {
		confirmed = true
		retErr = fmt.Errorf("invalid debt because not enough confirmed block number, wanted is %d, actual is %d", common.ConfirmedBlockNumber, duration)
	}

	return
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

// IfDebtPacked
// returns packed whether debt is packed
// returns confirmed whether debt is confirmed
// returns retErr this error is return when debt is found invalid. which means we need remove this debt.
func (manager *LightClientsManager) IfDebtPacked(debt *types.Debt) (packed bool, confirmed bool, retErr error) {
	toShard := debt.Data.Account.Shard()
	if toShard == 0 || toShard == manager.localShard {
		retErr = errWrongShardDebt
		return
	}

	backend := manager.lightClientsBackend[toShard]
	result, index, err := backend.GetDebt(debt.Hash)

	if err != nil {
		return
	}

	if index == nil {
		return
	}

	_, err = result.Validate(nil, false, toShard)
	if err != nil {
		retErr = errors.NewStackedError(err, "debt validate failed")
		return
	}

	packed = true

	// only marked as packed when the debt is confirmed
	header := backend.ChainBackend().CurrentHeader()
	if header.Height-index.BlockHeight >= common.ConfirmedBlockNumber {
		confirmed = true
	}

	return
}
