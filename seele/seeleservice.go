/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/seeleteam/go-seele/event"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/miner"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/p2p"
	rpc "github.com/seeleteam/go-seele/rpc2"
	"github.com/seeleteam/go-seele/seele/download"
)

const chainHeaderChangeBuffSize = 100

// SeeleService implements full node service.
type SeeleService struct {
	networkID     uint64
	p2pServer     *p2p.Server
	seeleProtocol *SeeleProtocol
	log           *log.SeeleLog

	txPool         *core.TransactionPool
	debtPool       *core.DebtPool
	chain          *core.Blockchain
	chainDB        database.Database // database used to store blocks.
	accountStateDB database.Database // database used to store account state info.
	miner          *miner.Miner

	lastHeader               common.Hash
	chainHeaderChangeChannel chan common.Hash
}

// ServiceContext is a collection of service configuration inherited from node
type ServiceContext struct {
	DataDir string
}

func (s *SeeleService) BlockChain() *core.Blockchain      { return s.chain }
func (s *SeeleService) Miner() *miner.Miner               { return s.miner }
func (s *SeeleService) AccountStateDB() database.Database { return s.accountStateDB }
func (s *SeeleService) Downloader() *downloader.Downloader {
	return s.seeleProtocol.Downloader()
}

// NewSeeleService create SeeleService
func NewSeeleService(ctx context.Context, conf *node.Config, log *log.SeeleLog) (s *SeeleService, err error) {
	s = &SeeleService{
		log:       log,
		networkID: conf.P2PConfig.NetworkID,
	}

	serviceContext := ctx.Value("ServiceContext").(ServiceContext)

	// Initialize blockchain DB.
	chainDBPath := filepath.Join(serviceContext.DataDir, BlockChainDir)
	log.Info("NewSeeleService BlockChain datadir is %s", chainDBPath)
	s.chainDB, err = leveldb.NewLevelDB(chainDBPath)
	if err != nil {
		log.Error("NewSeeleService Create BlockChain err. %s", err)
		return nil, err
	}
	leveldb.StartMetrics(s.chainDB, "chaindb", log)

	// Initialize account state info DB.
	accountStateDBPath := filepath.Join(serviceContext.DataDir, AccountStateDir)
	log.Info("NewSeeleService account state datadir is %s", accountStateDBPath)
	s.accountStateDB, err = leveldb.NewLevelDB(accountStateDBPath)
	if err != nil {
		s.chainDB.Close()
		log.Error("NewSeeleService Create BlockChain err: failed to create account state DB, %s", err)
		return nil, err
	}

	// initialize and validate genesis
	bcStore := store.NewCachedStore(store.NewBlockchainDatabase(s.chainDB))
	genesis := core.GetGenesis(conf.SeeleConfig.GenesisConfig)

	err = genesis.InitializeAndValidate(bcStore, s.accountStateDB)
	if err != nil {
		s.chainDB.Close()
		s.accountStateDB.Close()
		log.Error("NewSeeleService genesis.Initialize err. %s", err)
		return nil, err
	}

	recoveryPointFile := filepath.Join(serviceContext.DataDir, BlockChainRecoveryPointFile)
	s.chain, err = core.NewBlockchain(bcStore, s.accountStateDB, recoveryPointFile)
	if err != nil {
		s.chainDB.Close()
		s.accountStateDB.Close()
		log.Error("failed to init chain in NewSeeleService. %s", err)
		return nil, err
	}

	err = s.initPool(conf)
	if err != nil {
		s.chainDB.Close()
		s.accountStateDB.Close()
		log.Error("failed to create transaction pool in NewSeeleService, %s", err)
		return nil, err
	}

	s.seeleProtocol, err = NewSeeleProtocol(s, log)
	if err != nil {
		s.chainDB.Close()
		s.accountStateDB.Close()
		log.Error("failed to create seeleProtocol in NewSeeleService, %s", err)
		return nil, err
	}

	s.miner = miner.NewMiner(conf.SeeleConfig.Coinbase, s)

	return s, nil
}

func (s *SeeleService) initPool(conf *node.Config) error {
	var err error
	s.lastHeader, err = s.chain.GetStore().GetHeadBlockHash()
	if err != nil {
		return fmt.Errorf("failed to get chain header, %s", err)
	}

	s.chainHeaderChangeChannel = make(chan common.Hash, chainHeaderChangeBuffSize)
	s.debtPool = core.NewDebtPool(s.chain)
	s.txPool = core.NewTransactionPool(conf.SeeleConfig.TxConf, s.chain)

	event.ChainHeaderChangedEventMananger.AddAsyncListener(s.chainHeaderChanged)
	go s.MonitorChainHeaderChange()

	return nil
}

// chainHeaderChanged handle chain header changed event.
// add forked transaction back
// deleted invalid transaction
func (s *SeeleService) chainHeaderChanged(e event.Event) {
	newHeader := e.(common.Hash)
	if newHeader.IsEmpty() {
		return
	}

	s.chainHeaderChangeChannel <- newHeader
}

// MonitorChainHeaderChange monitor and handle chain header event
func (s *SeeleService) MonitorChainHeaderChange() {
	for {
		select {
		case newHeader := <-s.chainHeaderChangeChannel:
			if s.lastHeader.IsEmpty() {
				s.lastHeader = newHeader
				return
			}

			s.txPool.HandleChainHeaderChanged(newHeader, s.lastHeader)
			s.debtPool.HandleChainHeaderChanged(newHeader, s.lastHeader)

			s.lastHeader = newHeader
		}
	}
}

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *SeeleService) Protocols() (protos []p2p.Protocol) {
	protos = append(protos, s.seeleProtocol.Protocol)
	return
}

// Start implements node.Service, starting goroutines needed by SeeleService.
func (s *SeeleService) Start(srvr *p2p.Server) error {
	s.p2pServer = srvr

	s.seeleProtocol.Start()
	return nil
}

// Stop implements node.Service, terminating all internal goroutines.
func (s *SeeleService) Stop() error {
	s.seeleProtocol.Stop()

	//TODO
	// s.txPool.Stop() s.chain.Stop()
	// retries? leave it to future
	s.chainDB.Close()
	s.accountStateDB.Close()
	return nil
}

// APIs implements node.Service, returning the collection of RPC services the seele package offers.
func (s *SeeleService) APIs() (apis []rpc.API) {
	apis = append(apis, api.GetAPIs(s)...)
	return append(apis, []rpc.API{
		{
			Namespace: "seele",
			Version:   "1.0",
			Service:   NewPublicSeeleAPI(s),
			Public:    true,
		},
		{
			Namespace: "download",
			Version:   "1.0",
			Service:   downloader.NewPrivatedownloaderAPI(s.seeleProtocol.downloader),
			Public:    false,
		},
		{
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s),
			Public:    false,
		},
		{
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		},
	}...)
}
