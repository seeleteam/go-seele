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
	Long:  `Create a new account with the balance(default is 0)`,
	Run: func(cmd *cobra.Command, args []string) {
		statedb, _, dispose, err := preprocessContract()
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		defer dispose()

		// Generate a non-existent address
		addr := *crypto.MustGenerateRandomAddress()
		for {
			if !statedb.Exist(addr) {
				break
			}
			addr = *crypto.MustGenerateRandomAddress()
		}

		statedb.CreateAccount(addr)
		statedb.SetBalance(addr, new(big.Int).SetUint64(balance))
		statedb.SetNonce(addr, DefaultNonce)

		fmt.Println("The new account address is ", addr.ToHex())
	},
}

var setCmd = &cobra.Command{
	Use:   "set",
	Short: "set the balance of the account",
	Long:  `set the balance(default is 0) of the account`,
	Run: func(cmd *cobra.Command, args []string) {
		statedb, _, dispose, err := preprocessContract()
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		defer dispose()

		addr, err := common.HexToAddress(account)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		if balance >= 0 {
			statedb.SetBalance(addr, new(big.Int).SetUint64(balance))
		}

		fmt.Println("Set the balance successful, the balance of the account is ", balance)
	},
}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "get the balance of the account",
	Long:  `get the balance of the account, if the account is non-existence, return 0`,
	Run: func(cmd *cobra.Command, args []string) {
		statedb, _, dispose, err := preprocessContract()
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		defer dispose()

		addr, err := common.HexToAddress(account)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		fmt.Println("The balance of the account is ", statedb.GetBalance(addr))
	},
}
