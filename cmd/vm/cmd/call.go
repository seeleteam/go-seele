/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

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
	contractHexAddr string
	input           string
)

func init() {
	callCmd.Flags().StringVarP(&input, "input", "i", "", "call function input(Required)")
	callCmd.MarkFlagRequired("input")

	callCmd.Flags().StringVarP(&contractHexAddr, "contractAddr", "c", "", "the contract address")

	callCmd.Flags().StringVarP(&account, "account", "a", "", "invoking the address of calling the smart contract(Default is random and has 100 balance)")
	rootCmd.AddCommand(callCmd)
}

var callCmd = &cobra.Command{
	Use:   "call",
	Short: "call a contract",
	Long:  `All contract could callable. This is Seele contract simulator's`,
	Run: func(cmd *cobra.Command, args []string) {
		callContract(contractHexAddr, input)
	},
}

func callContract(contractHexAddr string, input string) {
	db, statedb, bcStore, dispose, err := preprocessContract()
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

	// Contract address
	var contractAddr common.Address
	if len(contractHexAddr) > 0 {
		if contractAddr, err = common.HexToAddress(contractHexAddr); err != nil {
			fmt.Println("Invalid contract address,", err.Error())
			return
		}
	} else if contractAddr = getGlobalContractAddress(db); contractAddr.IsEmpty() {
		fmt.Println("Contract address not specified.")
		return
	}

	// Call method and input parameters
	msg, err := hexutil.HexToBytes(input)
	if err != nil {
		fmt.Println("Invalid input message,", err.Error())
		return
	}

	// Create a call message transaction
	callContractTx, err := types.NewMessageTransaction(from, contractAddr, big.NewInt(0), big.NewInt(0), DefaultNonce, msg)
	if err != nil {
		fmt.Println("Failed to create message tx,", err.Error())
		return
	}

	receipt, err := processContract(statedb, bcStore, callContractTx)
	if err != nil {
		fmt.Println("Failed to call contract,", err.Error())
		return
	}

	// Print the result
	fmt.Println()
	fmt.Println("Succeed to call contract!")

	if len(receipt.Result) > 0 {
		fmt.Println("Result (raw):", receipt.Result)
		fmt.Println("Result (hex):", hexutil.BytesToHex(receipt.Result))
		fmt.Println("Result (big):", new(big.Int).SetBytes(receipt.Result))
	}

	for i, log := range receipt.Logs {
		fmt.Printf("Log[%v]:\n", i)
		fmt.Println("\taddress:", log.Address.ToHex())
		if len(log.Topics) == 1 {
			fmt.Println("\ttopics:", log.Topics[0].ToHex())
		} else {
			fmt.Println("\ttopics:", log.Topics)
		}
		dataLen := len(log.Data) / 32
		for i := 0; i < dataLen; i++ {
			fmt.Printf("\tdata[%v]: %v\n", i, log.Data[i*32:i*32+32])
		}
	}
}
