/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"math"
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
	"github.com/spf13/cobra"
)

var (
	contractHexAddr string
	input           string
	methodName      string
)

func init() {
	callCmd.Flags().StringVarP(&input, "input", "i", "", "call function input")
	callCmd.Flags().StringVarP(&methodName, "method", "m", "", "call function method name")
	callCmd.Flags().StringVarP(&contractHexAddr, "contractAddr", "c", "", "the contract address")
	callCmd.Flags().StringVarP(&account, "account", "a", "", "invoking the address of calling the smart contract(Default is random and has 1 seele)")
	rootCmd.AddCommand(callCmd)
}

var callCmd = &cobra.Command{
	Use:   "call",
	Short: "call a contract",
	Long:  `All contract could callable. This is Seele contract simulator's`,
	Run: func(cmd *cobra.Command, args []string) {
		callContract(contractHexAddr)
	},
}

func callContract(contractHexAddr string) {
	db, statedb, bcStore, dispose, err := preprocessContract()
	if err != nil {
		fmt.Println("failed to prepare the simulator environment,", err.Error())
		return
	}
	defer dispose()

	// Get the invoking address
	from := getFromAddress(statedb)
	if from.IsEmpty() {
		return
	}

	// Contract address
	contractAddr := getContractAddress(db)
	if contractAddr.IsEmpty() {
		return
	}

	// Input message to call contract
	input := getContractInputMsg(db, contractAddr.Bytes())
	if len(input) == 0 {
		return
	}

	// Call method and input parameters
	msg, err := hexutil.HexToBytes(input)
	if err != nil {
		fmt.Println("Invalid input message,", err.Error())
		return
	}

	// Create a call message transaction
	callContractTx, err := types.NewMessageTransaction(from, contractAddr, big.NewInt(0), big.NewInt(1), math.MaxUint64, DefaultNonce, msg)
	if err != nil {
		fmt.Println("failed to create message tx,", err.Error())
		return
	}

	receipt, err := processContract(statedb, bcStore, callContractTx)
	if err != nil {
		fmt.Println("failed to call contract,", err.Error())
		return
	}

	// Print the result
	fmt.Println()
	fmt.Println("contract called successfully")

	if len(receipt.Result) > 0 {
		fmt.Println("Result (raw):", receipt.Result)
		fmt.Println("Result (hex):", hexutil.BytesToHex(receipt.Result))
		fmt.Println("Result (big):", new(big.Int).SetBytes(receipt.Result))
	}

	for i, log := range receipt.Logs {
		fmt.Printf("Log[%v]:\n", i)
		fmt.Println("\taddress:", log.Address.Hex())
		if len(log.Topics) == 1 {
			fmt.Println("\ttopics:", log.Topics[0].Hex())
		} else {
			fmt.Println("\ttopics:", log.Topics)
		}
		dataLen := len(log.Data) / 32
		for i := 0; i < dataLen; i++ {
			fmt.Printf("\tdata[%v]: %v\n", i, log.Data[i*32:i*32+32])
		}
	}
}

func getContractAddress(db database.Database) common.Address {
	if len(contractHexAddr) == 0 {
		addr := getGlobalContractAddress(db)
		if addr.IsEmpty() {
			fmt.Println("Contract address not specified.")
		}

		return addr
	}

	addr, err := common.HexToAddress(contractHexAddr)
	if err != nil {
		fmt.Println("Invalid contract address,", err.Error())
		return common.EmptyAddress
	}

	return addr
}

func getContractInputMsg(db database.Database, contractAddr []byte) string {
	if len(input) > 0 {
		return input
	}

	if len(methodName) == 0 {
		fmt.Println("Input or method not specified.")
		return ""
	}

	output := getContractCompilationOutput(db, contractAddr)
	if output == nil {
		fmt.Println("Cannot find the contract info in DB.")
		return ""
	}

	method := output.getMethodByName(methodName)
	if method == nil {
		return ""
	}

	return method.createInput()
}
