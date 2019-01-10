//////////////////////////////////////////////
//This is a tool for analysis all information
//1. trasactions: sent, processed, pending
//2. ...
//3. ...
//////////////////////////////////////////////
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// var (
// 	//txs already sent
// 	txSent int
// 	//txs already in block
// 	txBlock int
// 	//txs pending
// 	txPool int
// )

var txAnalysisCmd = &cobra.Command{
	Use:   "txAnalysis",
	Short: "check whether any tx gets lost. RUN ONLY AFTER TPS = 0",
	Long: `For Example:
  		   tool.exe txAnalysis`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Have you stopped sendtx and TPS = 0 (y/n)?")
		confirm := askForConfirmation()
		if confirm == true {
			initClient()
			//get the txs already minned on blockchain:
			for {
				var height uint64
				var counter uint64
				var txCount int
				txCount = 0

				var blockTxCount int
				for clientIndex := range clientList {
					if err := clientList[clientIndex].Call(&height, "seele_getBlockHeight"); err != nil {
						panic(fmt.Sprintf("failed to get the block height: %s", err))
					}
					fmt.Printf("block height %d\n", height)
					counter = 1
					// get the tx count up to current block height
					for counter <= height {
						blockTxCount = 0
						if err := clientList[clientIndex].Call(&blockTxCount, "txpool_getBlockTransactionCount", "", counter); err != nil {
							panic(fmt.Sprintf("failed to get the block tx count: %s\n", err))
						}
						txCount += blockTxCount - 1 //first tx is reward to miner
						counter++
					}

				}
				//fmt.Printf("Tx Sent: %d\n", txTotal)
				fmt.Printf("Tx Processed: %d\n", txCount)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(txAnalysisCmd)
}
