/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"context"
	"path/filepath"

	"github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/seeleteam/go-seele/seele"
)

// ServiceClient implements service for light mode.
type ServiceClient struct {
	networkID     string
	netVersion    string
	p2pServer     *p2p.Server
	seeleProtocol *LightProtocol
	log           *log.SeeleLog
	odrBackend    *odrBackend

	txPool  *txPool
	chain   *LightChain
	lightDB database.Database // database used to store blocks and account state.

	shard uint
}

// NewServiceClient create ServiceClient
func NewServiceClient(ctx context.Context, conf *node.Config, log *log.SeeleLog, dbFolder string, shard uint, engine consensus.Engine) (s *ServiceClient, err error) {
	s = &ServiceClient{
		log:        log,
		networkID:  conf.P2PConfig.NetworkID,
		netVersion: conf.BasicConfig.Version,
		shard:      shard,
	}

	serviceContext := ctx.Value("ServiceContext").(seele.ServiceContext)
	// Initialize blockchain DB.
	chainDBPath := filepath.Join(serviceContext.DataDir, dbFolder)
	log.Info("NewServiceClient BlockChain datadir is %s", chainDBPath)
	s.lightDB, err = leveldb.NewLevelDB(chainDBPath)
	if err != nil {
		log.Error("NewServiceClient Create lightDB err. %s", err)
		return nil, err
	}

	bcStore := store.NewCachedStore(store.NewBlockchainDatabase(s.lightDB))
	s.odrBackend = newOdrBackend(bcStore, shard)
	// initialize and validate genesis
	genesis := core.GetGenesis(&conf.SeeleConfig.GenesisConfig)

	err = genesis.InitializeAndValidate(bcStore, s.lightDB)
	if err != nil {
		s.lightDB.Close()
		s.odrBackend.close()
		log.Error("NewServiceClient genesis.Initialize err. %s", err)
		return nil, err
	}

	s.chain, err = newLightChain(bcStore, s.lightDB, s.odrBackend, engine)
	if err != nil {
		s.lightDB.Close()
		s.odrBackend.close()
		log.Error("failed to init chain in NewServiceClient. %s", err)
		return nil, err
	}

	s.txPool = newTxPool(s.chain, s.odrBackend, s.chain.headerChangedEventManager)

	s.seeleProtocol, err = NewLightProtocol(conf.P2PConfig.NetworkID, s.txPool, nil, s.chain, false, s.odrBackend, log, shard)
	if err != nil {
		s.lightDB.Close()
		s.odrBackend.close()
		log.Error("failed to create seeleProtocol in NewServiceClient, %s", err)
		return nil, err
	}

	s.odrBackend.start(s.seeleProtocol.peerSet)
	log.Info("Light mode started.")
	return s, nil
}

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *ServiceClient) Protocols() (protos []p2p.Protocol) {
	return append(protos, s.seeleProtocol.Protocol)
}

// Start implements node.Service, starting goroutines needed by ServiceClient.
func (s *ServiceClient) Start(srvr *p2p.Server) error {
	s.p2pServer = srvr

	s.seeleProtocol.Start()
	return nil
}

// Stop implements node.Service, terminating all internal goroutines.
func (s *ServiceClient) Stop() error {
	s.seeleProtocol.Stop()
	s.lightDB.Close()
	s.odrBackend.close()
	return nil
}

// APIs implements node.Service, returning the collection of RPC services the seele package offers.
func (s *ServiceClient) APIs() (apis []rpc.API) {
	return append(apis, api.GetAPIs(NewLightBackend(s))...)
}
