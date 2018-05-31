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
	callCmd.Flags().StringVarP(&contractAddr, "contractAddr", "c", "", "the contract address")
	callCmd.Flags().StringVarP(&input, "input", "i", "", "call function input")
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
		fmt.Println(err.Error())
		return
	}

	// Call method and input parameters
	msg, err := hexutil.HexToBytes(input)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	statedb, bcStore, dispose, err := preprocessContract()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer dispose()

	// Create a call message transaction
	callContractTx, err := types.NewMessageTransaction(*crypto.MustGenerateRandomAddress(), contract, big.NewInt(0), big.NewInt(0), DefaultNonce, msg)

	receipt, err := processContract(statedb, bcStore, callContractTx)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Print the result
	fmt.Println("Contract call success, The result: ", receipt.Result)
}
