// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"net/rpc/jsonrpc"
	"github.com/seeleteam/go-seele/common"
)

// getcoinbaseCmd represents the getcoinbase command
var getcoinbaseCmd = &cobra.Command{
	Use:   "getcoinbase",
	Short: "get coinbase address",
	Long: `get coinbase address
		For example:
			client.exe getcoinbase -a 127.0.0.1:55027`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("getcoinbase called")

		client, err := jsonrpc.Dial("tcp", rpcAddr)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer client.Close()

		addr := new(common.Address)
		err = client.Call("seele.Coinbase", nil, addr)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("coinbase address: %v\n", addr.ToHex())
	},
}

func init() {
	rootCmd.AddCommand(getcoinbaseCmd)
}
