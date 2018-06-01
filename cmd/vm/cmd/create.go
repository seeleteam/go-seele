package cmd

import (
	"fmt"
	"math/big"
	"path/filepath"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/spf13/cobra"
)

var (
	dir        string
	account    string
	code       string
	defaultDir string
)

func init() {
	createCmd.Flags().StringVarP(&code, "create", "c", "", "create a contract")
	defaultDir = filepath.Join(common.GetDefaultDataFolder(), "simulator")
	createCmd.Flags().StringVarP(&dir, "directory", "d", defaultDir, "test directory(default is $HOME/.seele/simulator)")
	createCmd.Flags().StringVarP(&account, "account", "a", "", "the account address(default is random and has 100 eth)")
	rootCmd.AddCommand(createCmd)
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "create a contract",
	Long:  `All contract simulators can create contracts. This is Seele contract simulator's`,
	Run: func(cmd *cobra.Command, args []string) {
		createContract()
	},
}

func createContract() {
	// Binary contract code
	bytecode, err := hexutil.HexToBytes(code)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Get a directory to save the contract
	if dir != defaultDir {
		fmt.Println("Now the directory flag is unused")
		return
	}

	statedb, bcStore, dispose, err := preprocessContract()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer dispose()

	// Get an account to create the contract
	var from common.Address
	if account == "" {
		from = *crypto.MustGenerateRandomAddress()
		statedb.CreateAccount(from)
		statedb.SetBalance(from, new(big.Int).SetUint64(100))
		statedb.SetNonce(from, DefaultNonce)
	} else {
		fmt.Println("Now the account flag is unused")
		return
	}

	// Create a contract
	createContractTx, err := types.NewContractTransaction(from, big.NewInt(0), big.NewInt(0), DefaultNonce, bytecode)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	receipt, err := processContract(statedb, bcStore, createContractTx)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Print the contract Address
	fmt.Println("Contract creation is completed! The contract address: ", hexutil.BytesToHex(receipt.ContractAddress))
}
