/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/seeleteam/go-seele/api"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/seeleteam/go-seele/event"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/miner"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/seeleteam/go-seele/seele/download"
)

const chainHeaderChangeBuffSize = 100

// SeeleService implements full node service.
type SeeleService struct {
	networkID     string
	netVersion    string
	p2pServer     *p2p.Server
	seeleProtocol *SeeleProtocol
	log           *log.SeeleLog

	txPool             *core.TransactionPool
	debtPool           *core.DebtPool
	chain              *core.Blockchain
	chainDB            database.Database // database used to store blocks.
	chainDBPath        string
	accountStateDB     database.Database // database used to store account state info.
	accountStateDBPath string
	miner              *miner.Miner

	lastHeader               common.Hash
	chainHeaderChangeChannel chan common.Hash

	debtVerifier types.DebtVerifier
}

// ServiceContext is a collection of service configuration inherited from node
type ServiceContext struct {
	DataDir string
}

// AccountStateDB return account state db
func (s *SeeleService) AccountStateDB() database.Database { return s.accountStateDB }

// BlockChain get blockchain
func (s *SeeleService) BlockChain() *core.Blockchain { return s.chain }

// TxPool tx pool
func (s *SeeleService) TxPool() *core.TransactionPool { return s.txPool }

// DebtPool debt pool
func (s *SeeleService) DebtPool() *core.DebtPool { return s.debtPool }

// NetVersion net version
func (s *SeeleService) NetVersion() string { return s.netVersion }

// NetWorkID net id
func (s *SeeleService) NetWorkID() string { return s.networkID }

// Miner get miner
func (s *SeeleService) Miner() *miner.Miner { return s.miner }

// Downloader get downloader
func (s *SeeleService) Downloader() *downloader.Downloader {
	return s.seeleProtocol.Downloader()
}

// NewSeeleService create SeeleService
func NewSeeleService(ctx context.Context, conf *node.Config, log *log.SeeleLog, engine consensus.Engine, verifier types.DebtVerifier) (s *SeeleService, err error) {
	s = &SeeleService{
		log:          log,
		networkID:    conf.P2PConfig.NetworkID,
		netVersion:   conf.BasicConfig.Version,
		debtVerifier: verifier,
	}

	serviceContext := ctx.Value("ServiceContext").(ServiceContext)

	// Initialize blockchain DB.
	if err = s.initBlockchainDB(&serviceContext); err != nil {
		return nil, err
	}

	leveldb.StartMetrics(s.chainDB, "chaindb", log)

	// Initialize account state info DB.
	if err = s.initAccountStateDB(&serviceContext); err != nil {
		return nil, err
	}

	s.miner = miner.NewMiner(conf.SeeleConfig.Coinbase, s, s.debtVerifier, engine)

	// initialize and validate genesis
	if err = s.initGenesisAndChain(&serviceContext, conf); err != nil {
		return nil, err
	}

	if err = s.initPool(conf); err != nil {
		return nil, err
	}

	if s.seeleProtocol, err = NewSeeleProtocol(s, log); err != nil {
		s.Stop()
		log.Error("failed to create seeleProtocol in NewSeeleService, %s", err)
		return nil, err
	}

	return s, nil
}

func (s *SeeleService) initBlockchainDB(serviceContext *ServiceContext) (err error) {
	s.chainDBPath = filepath.Join(serviceContext.DataDir, BlockChainDir)
	s.log.Info("NewSeeleService BlockChain datadir is %s", s.chainDBPath)

	if s.chainDB, err = leveldb.NewLevelDB(s.chainDBPath); err != nil {
		s.log.Error("NewSeeleService Create BlockChain err. %s", err)
		return err
	}

	return nil
}

func (s *SeeleService) initAccountStateDB(serviceContext *ServiceContext) (err error) {
	s.accountStateDBPath = filepath.Join(serviceContext.DataDir, AccountStateDir)
	s.log.Info("NewSeeleService account state datadir is %s", s.accountStateDBPath)

	if s.accountStateDB, err = leveldb.NewLevelDB(s.accountStateDBPath); err != nil {
		s.Stop()
		s.log.Error("NewSeeleService Create BlockChain err: failed to create account state DB, %s", err)
		return err
	}

	return nil
}

func (s *SeeleService) initGenesisAndChain(serviceContext *ServiceContext, conf *node.Config) (err error) {
	bcStore := store.NewCachedStore(store.NewBlockchainDatabase(s.chainDB))
	genesis := core.GetGenesis(&conf.SeeleConfig.GenesisConfig)

	if err = genesis.InitializeAndValidate(bcStore, s.accountStateDB); err != nil {
		s.Stop()
		s.log.Error("NewSeeleService genesis.Initialize err. %s", err)
		return err
	}

	recoveryPointFile := filepath.Join(serviceContext.DataDir, BlockChainRecoveryPointFile)
	if s.chain, err = core.NewBlockchain(bcStore, s.accountStateDB, recoveryPointFile, s.miner.GetEngine(), s.debtVerifier); err != nil {
		s.Stop()
		s.log.Error("failed to init chain in NewSeeleService. %s", err)
		return err
	}

	return nil
}

func (s *SeeleService) initPool(conf *node.Config) (err error) {
	if s.lastHeader, err = s.chain.GetStore().GetHeadBlockHash(); err != nil {
		s.Stop()
		return fmt.Errorf("failed to get chain header, %s", err)
	}

	s.chainHeaderChangeChannel = make(chan common.Hash, chainHeaderChangeBuffSize)
	s.debtPool = core.NewDebtPool(s.chain, s.debtVerifier)
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
	return protos
}

// Start implements node.Service, starting goroutines needed by SeeleService.
func (s *SeeleService) Start(srvr *p2p.Server) error {
	s.p2pServer = srvr
	s.seeleProtocol.Start()

	return nil
}

// Stop implements node.Service, terminating all internal goroutines.
func (s *SeeleService) Stop() error {
	//TODO
	// s.txPool.Stop() s.chain.Stop()
	// retries? leave it to future
	if s.seeleProtocol != nil {
		s.seeleProtocol.Stop()
		s.seeleProtocol = nil
	}

	if s.chainDB != nil {
		s.chainDB.Close()
		os.RemoveAll(s.chainDBPath)
		s.chainDB = nil
	}

	if s.accountStateDB != nil {
		s.accountStateDB.Close()
		os.RemoveAll(s.accountStateDBPath)
		s.accountStateDB = nil
	}

	return nil
}

// APIs implements node.Service, returning the collection of RPC services the seele package offers.
// must to make sure that the order of the download api is 5; we get the download api by 5
func (s *SeeleService) APIs() (apis []rpc.API) {
	apis = append(apis, api.GetAPIs(NewSeeleBackend(s))...)
	apis = append(apis, []rpc.API{
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
		{
			Namespace: "txpool",
			Version:   "1.0",
			Service:   NewTransactionPoolAPI(s),
			Public:    false,
		},
	}...)

	minerApis := s.miner.GetEngine().APIs(s.chain)
	apis = append(apis, minerApis...)

	return apis
}
