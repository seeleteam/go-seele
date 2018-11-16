/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"math/big"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/spf13/cobra"
)

var (
	balance uint64
)

func init() {
	// New account
	newCmd.Flags().Uint64VarP(&balance, "balance", "b", 0, "create a new account with the balance(Default is 0)")
	rootCmd.AddCommand(newCmd)

	// Set account
	setCmd.Flags().StringVarP(&account, "account", "a", "", "the account to set balance(Required)")
	setCmd.Flags().Uint64VarP(&balance, "balance", "b", 0, "the balance of the account to set(Required)")
	setCmd.MarkFlagRequired("account")
	setCmd.MarkFlagRequired("balance")
	rootCmd.AddCommand(setCmd)

	// Get account
	getCmd.Flags().StringVarP(&account, "account", "a", "", "the account to get balance(Required)")
	getCmd.MarkFlagRequired("account")
	rootCmd.AddCommand(getCmd)
}

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "create a new account",
	Long:  `Create a new account with the balance(Default is 0)`,
	Run: func(cmd *cobra.Command, args []string) {
		_, statedb, _, dispose, err := preprocessContract()
		if err != nil {
			fmt.Println("Failed to prepare the simulator environment,", err.Error())
			return
		}
		defer dispose()

		// Generate a random address
		addr := *crypto.MustGenerateRandomAddress()
		statedb.CreateAccount(addr)
		statedb.SetBalance(addr, new(big.Int).SetUint64(balance))
		statedb.SetNonce(addr, DefaultNonce)

		fmt.Println("The new account address is ", addr.Hex())
	},
}

var setCmd = &cobra.Command{
	Use:   "set",
	Short: "set the balance of the account",
	Long:  `Set the balance(Default is 0) of the account`,
	Run: func(cmd *cobra.Command, args []string) {
		_, statedb, _, dispose, err := preprocessContract()
		if err != nil {
			fmt.Println("Failed to prepare the simulator environment,", err.Error())
			return
		}
		defer dispose()

		addr, err := common.HexToAddress(account)
		if err != nil {
			fmt.Println("Invalid account address,", err.Error())
			return
		}

		if !statedb.Exist(addr) {
			fmt.Println("Input a non-existence account address ", account)
			return
		}

		// Update the balance of the account
		bigIntBalance := new(big.Int).SetUint64(balance)
		statedb.SetBalance(addr, bigIntBalance)

		fmt.Println("Set the balance successfully, the balance of the account is ", common.BigToDecimal(bigIntBalance.Mul(bigIntBalance, common.SeeleToFan)))
	},
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "get the balance of the account",
	Long:  `Get the balance of the account, if the account is non-existence, return 0`,
	Run: func(cmd *cobra.Command, args []string) {
		_, statedb, _, dispose, err := preprocessContract()
		if err != nil {
			fmt.Println("Failed to prepare the simulator environment,", err.Error())
			return
		}
		defer dispose()

		addr, err := common.HexToAddress(account)
		if err != nil {
			fmt.Println("Invalid account address,", err.Error())
			return
		}

		fmt.Println("The balance of the account is ", common.BigToDecimal(statedb.GetBalance(addr).Mul(statedb.GetBalance(addr), common.SeeleToFan)))
	},
}
