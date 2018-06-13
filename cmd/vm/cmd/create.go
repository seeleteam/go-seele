/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/spf13/cobra"
)

var (
	code    string
	solFile string
	account string

	defaultDir = filepath.Join(common.GetDefaultDataFolder(), "simulator")
)

type solCompileOutput struct {
	HexByteCodes    string
	FunctionHashMap map[string]string
}

func init() {
	createCmd.Flags().StringVarP(&code, "code", "c", "", "the binary code of the smart contract to create, or the name of a text file that contains the binary contract code in the local directory(Required)")
	createCmd.Flags().StringVarP(&solFile, "file", "f", "", "solidity file path")
	createCmd.Flags().StringVarP(&account, "account", "a", "", "the account address(Default is random and has 100 balance)")
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

func compileSol() (*solCompileOutput, func(), error) {
	if len(solFile) == 0 {
		return nil, nil, nil
	}

	if !common.FileOrFolderExists(solFile) {
		return nil, nil, fmt.Errorf("cannot find the specified solidity file %v", solFile)
	}

	// output to temp dir
	tempDir, err := ioutil.TempDir("", "SolCompile-")
	if err != nil {
		return nil, nil, err
	}
	deleteTempDir := true
	defer func() {
		if deleteTempDir {
			os.RemoveAll(tempDir)
		}
	}()

	// run solidity compilation command
	cmdArgs := fmt.Sprintf("--bin --hashes -o %v %v", tempDir, solFile)
	cmd := exec.Command("solc.exe", strings.Split(cmdArgs, " ")...)
	if err = cmd.Run(); err != nil {
		return nil, nil, err
	}

	// walk through the temp dir to construct compilation outputs
	output := new(solCompileOutput)
	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		content, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		switch filepath.Ext(path) {
		case ".signatures":
			buildSolCompileFuncHash(string(content), output)
		case ".bin":
			output.HexByteCodes = "0x" + string(content)
		}

		return nil
	}

	if err = filepath.Walk(tempDir, walkFunc); err != nil {
		return nil, nil, err
	}

	deleteTempDir = false

	return output, func() {
		os.RemoveAll(tempDir)
	}, nil
}

func buildSolCompileFuncHash(content string, output *solCompileOutput) {
	output.FunctionHashMap = make(map[string]string)
	shortNames := make(map[string]int)

	for _, line := range strings.Split(content, "\n") {
		if line = strings.Trim(line, "\r"); len(line) == 0 {
			continue
		}

		// add mapping: funcFullName <-> hash
		hash := "0x" + line[:8]
		funcFullName := string(line[10:])
		output.FunctionHashMap[funcFullName] = hash

		// add mapping: funcShortName <-> hash
		if idx := strings.IndexByte(funcFullName, '('); idx >= 0 {
			name := string(funcFullName[:idx])
			shortNames[name] = shortNames[name] + 1
			output.FunctionHashMap[name] = hash
		} else {
			panic("did not find the left bracket for the function name: " + line)
		}
	}

	// remove mapping for overloaded functions, in which case
	// user must specify the function full name to call a contract.
	for k, v := range shortNames {
		if v > 1 {
			delete(output.FunctionHashMap, k)
		}
	}
}

func createContract() {
	if len(solFile) == 0 && len(code) == 0 {
		fmt.Println("Code or solidity file not specified.")
		return
	}

	compileOutput, dispose, err := compileSol()
	if err != nil {
		fmt.Println("Failed to compile,", err.Error())
		return
	}
	defer dispose()

	if compileOutput != nil {
		code = compileOutput.HexByteCodes
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

	// Create a contract
	createContractTx, err := types.NewContractTransaction(from, big.NewInt(0), big.NewInt(0), DefaultNonce, bytecode)
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
	fmt.Println("Succeed to create contract!")
	fmt.Println("Contract address:", hexutil.BytesToHex(receipt.ContractAddress))

	setGlobalContractAddress(db, hexutil.BytesToHex(receipt.ContractAddress))

	if compileOutput != nil {
		setContractCompilationOutput(db, receipt.ContractAddress, compileOutput)
	}
}
