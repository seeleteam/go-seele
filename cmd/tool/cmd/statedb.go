/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"path/filepath"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/core/state"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/database"
	"github.com/seeleteam/go-seele/database/leveldb"
	"github.com/spf13/cobra"
)

const (
	dbNameStatedb = "AccountState"
)

var (
	rootDir               string
	numGenesisAccounts    int
	genesisAccountBalance uint64
	numBlocks             int
	numTxs                int
	numToAccounts         int

	statedbCmd = &cobra.Command{
		Use:   "statedb",
		Short: "statistic the disk utilization of statedb",
		Run: func(cmd *cobra.Command, args []string) {
			if err := statStatedb(); err != nil {
				log("failed to statistic the statedb utilization: %v", err)
				return
			}

			size := getLevelDBSize(rootDir, dbNameStatedb)
			fmt.Println("final leveldb size:", sizeToString(size))
		},
	}
)

func init() {
	rootCmd.AddCommand(statedbCmd)

	statedbCmd.Flags().StringVar(&rootDir, "root", "", "root folder of leveldb")
	rootCmd.MarkFlagRequired("root")
	statedbCmd.Flags().IntVarP(&numGenesisAccounts, "genesisAccounts", "a", 1, "number of accounts in genesis")
	statedbCmd.Flags().Uint64VarP(&genesisAccountBalance, "genesisAccountBalance", "b", math.MaxUint64, "account balance in genesis")
	statedbCmd.Flags().IntVar(&numBlocks, "blocks", 1, "number of blocks to write")
	statedbCmd.Flags().IntVar(&numTxs, "txs", 1, "number of txs to write in a block")
	statedbCmd.Flags().IntVarP(&numToAccounts, "toAccounts", "t", 0, "number of To accounts used in tx. If value <= 0, to address is new generated in each tx")
}

func statStatedb() error {
	// prepare leveldb folder
	if err := prepareDir(rootDir); err != nil {
		return errors.NewStackedErrorf(err, "failed to init root folder %v", rootDir)
	}

	// init leveldb
	db, err := leveldb.NewLevelDB(filepath.Join(rootDir, dbNameStatedb))
	if err != nil {
		return errors.NewStackedErrorf(err, "failed to create leveldb [%v]", dbNameStatedb)
	}
	defer db.Close()

	// prepare genessis
	root, genesisAccounts, err := prepareGenesis(db)
	if err != nil {
		return errors.NewStackedError(err, "failed to prepare genesis")
	}
	log("genesis prepared with %v accounts", numGenesisAccounts)

	// prepare to accounts
	var toAccounts []common.Address
	for i := 0; i < numToAccounts; i++ {
		toAccounts = append(toAccounts, *crypto.MustGenerateRandomAddress())
	}
	log("To accounts: %v", len(toAccounts))

	// writes data with statedb
	for i := 0; i < numBlocks; i++ {
		if root, err = writeStatedb(root, genesisAccounts, toAccounts, db); err != nil {
			return errors.NewStackedErrorf(err, "failed to write statedb[%v]", i+1)
		}

		size := getLevelDBSize(rootDir, dbNameStatedb)
		log("succeed to write statedb[%v]: %v", i+1, sizeToString(size))
	}

	return nil
}

func prepareGenesis(db database.Database) (common.Hash, []common.Address, error) {
	if numGenesisAccounts <= 0 {
		return common.EmptyHash, nil, errors.New("no accounts in genesis")
	}

	statedb := state.NewEmptyStatedb(db)
	var genesisAccounts []common.Address
	balance := new(big.Int).SetUint64(genesisAccountBalance)

	for i := 0; i < numGenesisAccounts; i++ {
		addr := *crypto.MustGenerateRandomAddress()
		statedb.CreateAccount(addr)
		statedb.AddBalance(addr, balance)
		genesisAccounts = append(genesisAccounts, addr)
	}

	root, err := commitStatedb(statedb, db)
	if err != nil {
		return common.EmptyHash, nil, errors.NewStackedError(err, "failed to commit genesis statedb")
	}

	return root, genesisAccounts, nil
}

func commitStatedb(statedb *state.Statedb, db database.Database) (common.Hash, error) {
	batch := db.NewBatch()

	root, err := statedb.Commit(batch)
	if err != nil {
		return common.EmptyHash, errors.NewStackedError(err, "failed to commit statedb changes to batch")
	}

	if err = batch.Commit(); err != nil {
		return common.EmptyHash, errors.NewStackedError(err, "failed to commit batch to leveldb")
	}

	return root, nil
}

func writeStatedb(root common.Hash, genesisAccounts []common.Address, toAccounts []common.Address, db database.Database) (common.Hash, error) {
	statedb, err := state.NewStatedb(root, db)
	if err != nil {
		return common.EmptyHash, errors.NewStackedErrorf(err, "failed to create statedb with root hash %v", root)
	}

	rand.Seed(time.Now().UnixNano())

	numFrom := len(genesisAccounts)
	numTo := len(toAccounts)
	if numTo > 0 && numTo < numTxs {
		return common.EmptyHash, errors.New("number of To address should not less than txs in a block")
	}

	toPerm := rand.Perm(numTo)

	for j := 0; j < numTxs; j++ {
		from := genesisAccounts[rand.Intn(numFrom)]
		statedb.SubBalance(from, big.NewInt(1))
		statedb.SetNonce(from, statedb.GetNonce(from)+1)

		var to common.Address
		if numTo == 0 {
			to = *crypto.MustGenerateRandomAddress()
		} else {
			to = toAccounts[toPerm[j]]
		}

		statedb.CreateAccount(to)
		statedb.AddBalance(to, big.NewInt(1))
	}

	if root, err = commitStatedb(statedb, db); err != nil {
		return common.EmptyHash, errors.NewStackedError(err, "failed to commit statedb")
	}

	return root, nil
}
