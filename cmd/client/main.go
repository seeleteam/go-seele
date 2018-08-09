/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"log"
	"os"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()

	minerCommands := cli.Command{
		Name:  "miner",
		Usage: "miner command",
		Subcommands: []cli.Command{
			{
				Name:  "start",
				Usage: "start miner",
				Flags: []cli.Flag{
					addressFlag,
					threadsFlag,
				},
				Action: RPCAction(StartMinerAction),
			},
			{
				Name:  "stop",
				Usage: "stop miner",
				Flags: []cli.Flag{
					addressFlag,
				},
				Action: RPCAction(StopMinerAction),
			},
			{
				Name:  "hashrate",
				Usage: "get miner hashrate",
				Flags: []cli.Flag{
					addressFlag,
				},
				Action: RPCAction(GetMinerHashrateAction),
			},
			{
				Name:  "getthreads",
				Usage: "get miner thread number",
				Flags: []cli.Flag{
					addressFlag,
				},
				Action: RPCAction(GetMinerThreadsAction),
			},
			{
				Name:  "setthreads",
				Usage: "set miner thread number",
				Flags: []cli.Flag{
					addressFlag,
					threadsFlag,
				},
				Action: RPCAction(SetMinerThreadsAction),
			},
			{
				Name:  "setcoinbase",
				Usage: "set miner coinbase",
				Flags: []cli.Flag{
					addressFlag,
					coinbaseFlag,
				},
				Action: RPCAction(SetMinerCoinbaseAction),
			},
			{
				Name:  "getcoinbase",
				Usage: "get miner coinbase",
				Flags: []cli.Flag{
					addressFlag,
				},
				Action: RPCAction(GetMinerCoinbaseAction),
			},
			{
				Name:  "status",
				Usage: "get miner status",
				Flags: []cli.Flag{
					addressFlag,
				},
				Action: RPCAction(GetMinerStatusAction),
			},
		},
	}

	p2pCommands := cli.Command{
		Name:  "p2p",
		Usage: "p2p commands",
		Subcommands: []cli.Command{
			{
				Name:  "peers",
				Usage: "get p2p peer connections",
				Flags: []cli.Flag{
					addressFlag,
				},
				Action: RPCAction(GetPeerCountAction),
			},
			{
				Name:  "peersinfo",
				Usage: "get p2p peers information",
				Flags: []cli.Flag{
					addressFlag,
				},
				Action: RPCAction(GetPeersInfo),
			},
			{
				Name:  "networkversion",
				Usage: "get current network version",
				Flags: []cli.Flag{
					addressFlag,
				},
				Action: RPCAction(GetNetworkVersion),
			},
			{
				Name:  "protocolversion",
				Usage: "get seele protocol version",
				Flags: []cli.Flag{
					addressFlag,
				},
				Action: RPCAction(GetProtocolVersion),
			},
		},
	}

	app.Commands = []cli.Command{
		minerCommands,
		p2pCommands,
		{
			Name:  "getinfo",
			Usage: "get node info",
			Flags: []cli.Flag{
				addressFlag,
			},
			Action: RPCAction(GetInfoAction),
		},
		{
			Name:  "getbalance",
			Usage: "get balance info",
			Flags: []cli.Flag{
				addressFlag,
				accountFlag,
			},
			Action: RPCAction(GetBalanceAction),
		},
		{
			Name:  "sendtx",
			Usage: "send transaction to node",
			Flags: []cli.Flag{
				addressFlag,
				fromFlag,
				toFlag,
				amountFlag,
				feeFlag,
				paloadFlag,
				nonceFlag,
			},
			Action: RPCAction(AddTxAction),
		},
		{
			Name:  "getnonce",
			Usage: "get account nonce",
			Flags: []cli.Flag{
				addressFlag,
				accountFlag,
			},
			Action: RPCAction(GetAccountNonceAction),
		},
		{
			Name:  "call",
			Usage: "call contract",
			Flags: []cli.Flag{
				addressFlag,
				fromFlag,
				toFlag,
				paloadFlag,
			},
			Action: RPCAction(CallAction),
		},
		{
			Name:  "getblockheight",
			Usage: "get block height",
			Flags: []cli.Flag{
				addressFlag,
			},
			Action: RPCAction(GetBlockHeightAction),
		},
		{
			Name:  "getblock",
			Usage: "get block by height or hash",
			Flags: []cli.Flag{
				addressFlag,
				heightFlag,
				hashFlag,
				fulltxFlag,
			},
			Action: RPCAction(GetBlockAction),
		},
		{
			Name:  "getlogs",
			Usage: "get logs",
			Flags: []cli.Flag{
				addressFlag,
				heightFlag,
				contractFlag,
				topicFlag,
			},
			Action: RPCAction(GetLogsAction),
		},
		{
			Name:  "gettxpoolcontent",
			Usage: "get transaction pool contents",
			Flags: []cli.Flag{
				addressFlag,
			},
			Action: RPCAction(GetTxPoolContentAction),
		},
		{
			Name:  "gettxpoolcount",
			Usage: "get transaction pool transaction count",
			Flags: []cli.Flag{
				addressFlag,
			},
			Action: RPCAction(GetTxPoolTxCountAction),
		},
		{
			Name:  "getblocktxcount",
			Usage: "get block transaction count by block height or block hash",
			Flags: []cli.Flag{
				addressFlag,
				heightFlag,
				hashFlag,
			},
			Action: RPCAction(GetBlockTransactionCountAction),
		},
		{
			Name:  "gettxinblock",
			Usage: "get transaction by block height or block hash with index of the transaction in the block",
			Flags: []cli.Flag{
				addressFlag,
				heightFlag,
				hashFlag,
				indexFlag,
			},
			Action: RPCAction(GetTransactionAction),
		},
		{
			Name:  "gettxbyhash",
			Usage: "get transaction by transaction hash",
			Flags: []cli.Flag{
				addressFlag,
				hashFlag,
			},
			Action: RPCAction(GetTransactionByHashAction),
		},
		{
			Name:  "getreceipt",
			Usage: "get receipt by transaction hash",
			Flags: []cli.Flag{
				addressFlag,
				hashFlag,
			},
			Action: RPCAction(GetReceiptByTxHashAction),
		},
		{
			Name:  "getpendingtxs",
			Usage: "get pending transactions",
			Flags: []cli.Flag{
				addressFlag,
			},
			Action: RPCAction(GetPendingTransactionsAction),
		},
		{
			Name:  "getshardnum",
			Usage: "get account shard number",
			Flags: []cli.Flag{
				accountFlag,
				privateKeyFlag,
			},
			Action: GetAccountShardNumAction,
		},
		{
			Name:  "savekey",
			Usage: "save private key to a keystore file",
			Flags: []cli.Flag{
				privateKeyFlag,
				fileNameFlag,
			},
			Action: SaveKeyAction,
		},
		{
			Name:  "sign",
			Usage: "generate a signed transaction and print it out",
			Flags: []cli.Flag{
				addressFlag,
				privateKeyFlag,
				toFlag,
				amountFlag,
				feeFlag,
				paloadFlag,
				nonceFlag,
			},
			Action: SignTxAction,
		},
		{
			Name:  "key",
			Usage: "generate key with or without shard number",
			Flags: []cli.Flag{
				shardFlag,
			},
			Action: GenerateKeyAction,
		},
		{
			Name:  "dumpheap",
			Usage: "dump heap for profiling, return the file path",
			Flags: []cli.Flag{
				addressFlag,
				gcBeforeDumpFlag,
				dumpFileFlag,
			},
			Action: RPCAction(GetDumpHeap),
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
