package cmd

import (
	"fmt"
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/spf13/cobra"
)

var (
	contractAddr string
	input        string
)

func init() {
	callCmd.Flags().StringVarP(&input, "input", "i", "", "call function input(Required)")
	callCmd.MarkFlagRequired("input")

	callCmd.Flags().StringVarP(&contractAddr, "contractAddr", "c", "", "the contract address(Required)")
	callCmd.MarkFlagRequired("contractAddr")

	callCmd.Flags().StringVarP(&account, "account", "a", "", "invoking the address of calling the smart contract(Default is random and has 100 balance)")
	rootCmd.AddCommand(callCmd)
}

var callCmd = &cobra.Command{
	Use:   "call",
	Short: "call a contract",
	Long:  `All contract could callable. This is Seele contract simulator's`,
	Run: func(cmd *cobra.Command, args []string) {
		callContract(contractAddr, input)
	},
}

func callContract(contractAddr string, input string) {
	// Contract address
	contract, err := common.HexToAddress(contractAddr)
	if err != nil {
		fmt.Println("Invalid contract address,", err.Error())
		return
	}

	// Call method and input parameters
	msg, err := hexutil.HexToBytes(input)
	if err != nil {
		fmt.Println("Invalid input message,", err.Error())
		return
	}

	statedb, bcStore, dispose, err := preprocessContract()
	if err != nil {
		fmt.Println("Failed to prepare the simulator environment,", err.Error())
		return
	}
	defer dispose()

	// Get the invoking address
	var from common.Address
	if account == "" {
		from = *crypto.MustGenerateRandomAddress()
		statedb.CreateAccount(from)
		statedb.SetBalance(from, new(big.Int).SetUint64(100))
		statedb.SetNonce(from, DefaultNonce)
	} else {
		from, err = common.HexToAddress(account)
		if err != nil {
			fmt.Println("Invalid account address,", err.Error())
			return
		}
	}

	// Create a call message transaction
	callContractTx, err := types.NewMessageTransaction(from, contract, big.NewInt(0), big.NewInt(0), DefaultNonce, msg)

	receipt, err := processContract(statedb, bcStore, callContractTx)
	if err != nil {
		fmt.Println("Failed to call contract,", err.Error())
		return
	}

	// Print the result
	fmt.Println()
	fmt.Println("Succeed to call contract!")
	fmt.Println("Result:", receipt.Result)
}
