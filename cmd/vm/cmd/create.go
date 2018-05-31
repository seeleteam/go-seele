package cmd

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/spf13/cobra"
)

var (
	dir     string
	account string
	code    string

	// NONCE default
	NONCE uint64 = 38
	// STATEDBHASH statedb hash
	STATEDBHASH = "WFF"
)

func init() {
	createCmd.Flags().StringVarP(&code, "create", "c", "", "create a contract")
	createCmd.Flags().StringVarP(&dir, "directory", "d", "$HOME/seele/monitor/", "test directory(default is $HOME/seele/monitor)")
	createCmd.Flags().StringVarP(&account, "account", "a", "", "the account address(default is random and has 100 eth)")
	rootCmd.AddCommand(createCmd)
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "vm createCmd",
	Long:  `All software has creates. This is vm's`,
	Run: func(cmd *cobra.Command, args []string) {
		createContract()
	},
}

func createContract() {
	// Binary contract code
	bytecode, err := hexutil.HexToBytes(code)
	if err != nil {
		panic(err)
	}

	// Get a directory to save the contract
	if dir != "$HOME/seele/monitor/" {
		panic(errors.New("Now the directory flag is unused"))
	}

	statedb, bcStore, dispose := preprocessContract()
	defer dispose()

	// Get an account to create the contract
	var from common.Address
	if account == "" {
		from = *crypto.MustGenerateRandomAddress()
		statedb.CreateAccount(from)
		statedb.SetBalance(from, new(big.Int).SetUint64(100))
		statedb.SetNonce(from, NONCE)
	} else {
		panic(errors.New("Now the account flag is unused"))
	}

	// Create a contract
	createContractTx, err := types.NewContractTransaction(from, big.NewInt(0), big.NewInt(0), NONCE, bytecode)
	if err != nil {
		panic(err)
	}

	receipt := processContract(statedb, bcStore, createContractTx)

	// Print the contract Address
	fmt.Println("Contract creation is completed! The contract address: ", hexutil.BytesToHex(receipt.ContractAddress))
}
