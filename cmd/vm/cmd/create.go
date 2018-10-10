/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"path/filepath"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/spf13/cobra"
)

var (
	code    string
	solFile string
	account string

	defaultDir = filepath.Join(common.GetDefaultDataFolder(), "simulator")
)

func init() {
	createCmd.Flags().StringVarP(&code, "code", "c", "", "the binary code of the smart contract to create, or the name of a readable file that contains the binary contract code in the local directory(Required)")
	createCmd.Flags().StringVarP(&solFile, "file", "f", "", "solidity file path")
	createCmd.Flags().StringVarP(&account, "account", "a", "", "the account address(Default is random and has 1 seele)")
	rootCmd.AddCommand(createCmd)
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "create a contract",
	Long:  "Create a contract with specified bytecodes or compiled bytecodes from specified solidity file.",
	Run: func(cmd *cobra.Command, args []string) {
		createContract()
	},
}

func createContract() {
	if len(solFile) == 0 && len(code) == 0 {
		fmt.Println("Code or solidity file not specified.")
		return
	}

	// compile solidity file if specified.
	var compileOutput *solCompileOutput
	if len(solFile) > 0 {
		output, dispose := compile(solFile)
		if output == nil {
			return
		}

		compileOutput = output
		code = output.HexByteCodes
		defer dispose()
	}

	// Try to read the file, if successful, use the file code
	if bytecode, err := ioutil.ReadFile(code); err == nil {
		code = string(bytecode)
	}

	bytecode, err := hexutil.HexToBytes(code)
	if err != nil {
		fmt.Println("Invalid code format,", err.Error())
		return
	}

	db, statedb, bcStore, dispose, err := preprocessContract()
	if err != nil {
		fmt.Println("Failed to prepare the simulator environment,", err.Error())
		return
	}
	defer dispose()

	// Get an account to create the contract
	from := getFromAddress(statedb)
	if from.IsEmpty() {
		return
	}

	// Create a contract
	createContractTx, err := types.NewContractTransaction(from, big.NewInt(0), big.NewInt(1), math.MaxUint64, DefaultNonce, bytecode)
	if err != nil {
		fmt.Println("Failed to create contract tx,", err.Error())
		return
	}

	receipt, err := processContract(statedb, bcStore, createContractTx)
	if err != nil {
		fmt.Println("Failed to create contract,", err.Error())
		return
	}

	// Print the contract Address
	fmt.Println()
	fmt.Println("contract created successfully")
	fmt.Println("Contract address:", hexutil.BytesToHex(receipt.ContractAddress))

	// Save contract info
	setGlobalContractAddress(db, hexutil.BytesToHex(receipt.ContractAddress))

	if compileOutput != nil {
		setContractCompilationOutput(db, receipt.ContractAddress, compileOutput)
	}
}
