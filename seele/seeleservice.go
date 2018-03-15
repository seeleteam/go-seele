/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/core/store"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/rpc"
)

// SeeleService implements full node service.
type SeeleService struct {
	networkID     uint64
	seeleProtocol *SeeleProtocol
	log           *log.SeeleLog
	coinbase      common.Address // account address that mining rewards will be send to.

	txPool  *core.TransactionPool
	chain   *core.Blockchain
	chainDB database.Database
}

func (s *SeeleService) TxPool() *core.TransactionPool { return s.txPool }
func (s *SeeleService) BlockChain() *core.Blockchain  { return s.chain }
func (s *SeeleService) NetVersion() uint64            { return s.networkID }

// ApplyTransaction applys a transaction
func (s *SeeleService) ApplyTransaction(coinbase common.Address, tx *types.Transaction) error {
	// TODO
	return nil
}

// NewSeeleService create SeeleService
func NewSeeleService(conf *Config, log *log.SeeleLog) (s *SeeleService, err error) {
	s = &SeeleService{
		networkID: conf.NetworkID,
		log:       log,
	}
	s.coinbase = conf.Coinbase
	dbPath := conf.DataRoot + BlockChainDir
	log.Info("NewSeeleService BlockChain datadir is %s", dbPath)
	s.chainDB, err = leveldb.NewLevelDB(dbPath)
	if err != nil {
		log.Error("NewSeeleService Create BlockChain err. %s", err)
		return nil, err
	}

	bcStore := store.NewBlockchainDatabase(s.chainDB)
	genesis := core.DefaultGenesis(bcStore)
	err = genesis.Initialize()
	if err != nil {
		log.Error("NewSeeleService genesis.Initialize err. %s", err)
		return nil, err
	}

	s.chain, err = core.NewBlockchain(bcStore)
	if err != nil {
		s.chainDB.Close()
		log.Error("NewSeeleService init chain failed. %s", err)
		return nil, err
	}

	s.txPool = core.NewTransactionPool(conf.txConf)
	s.seeleProtocol, err = NewSeeleProtocol(s, log)
	if err != nil {
		s.chainDB.Close()
		log.Error("NewSeeleService create seeleProtocol err. %s", err)
		return nil, err
	}

	return s, nil
}

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *SeeleService) Protocols() (protos []p2p.ProtocolInterface) {
	protos = append(protos, s.seeleProtocol)
	return
}

// Start implements node.Service, starting goroutines needed by SeeleService.
func (s *SeeleService) Start(srvr *p2p.Server) error {

	return nil
}

// Stop implements node.Service, terminating all internal goroutines.
func (s *SeeleService) Stop() error {
	s.seeleProtocol.Stop()

	//TODO
	// s.txPool.Stop() s.chain.Stop()
	s.chainDB.Close()
	return nil
}

// APIs implements node.Service, returning the collection of RPC services the seele package offers.
func (s *SeeleService) APIs() (apis []rpc.API) {
	//TODO add other api interface, for example consensus engine
	return append(apis, []rpc.API{
		{
			Namespace: "seele",
			Version:   "1.0",
			Service:   NewPublicSeeleAPI(s),
			Public:    true,
		},
	}...)
}
