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
	Long:  `All contract could callable. This is seele-vm's`,
	Run: func(cmd *cobra.Command, args []string) {
		callContract(contractAddr, input)
	},
}

func callContract(contractAddr string, input string) {
	// Contract address
	contract, err := common.HexToAddress(contractAddr)
	if err != nil {
		panic(err)
	}

	// Call method and input parameters
	msg, err := hexutil.HexToBytes(input)
	if err != nil {
		panic(err)
	}

	statedb, bcStore, dispose := preprocessContract()
	defer dispose()

	// Create a call message trascation
	callContractTx, err := types.NewMessageTransaction(*crypto.MustGenerateRandomAddress(), contract, big.NewInt(0), big.NewInt(0), NONCE, msg)

	receipt := processContract(statedb, bcStore, callContractTx)

	// Print the result
	fmt.Println("Contract call success, The result: ", string(receipt.Result))
}
