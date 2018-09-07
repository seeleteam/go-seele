/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"log"
	"os"
	"sort"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "client"
	app.Usage = "interact with node process"
	app.HideVersion = true

	minerCommands := cli.Command{
		Name:  "miner",
		Usage: "miner commands",
		Subcommands: []cli.Command{
			{
				Name:   "start",
				Usage:  "start miner",
				Flags:  rpcFlags(threadsFlag),
				Action: rpcAction("miner", "start"),
			},
			{
				Name:   "stop",
				Usage:  "stop miner",
				Flags:  rpcFlags(),
				Action: rpcAction("miner", "stop"),
			},
			{
				Name:   "hashrate",
				Usage:  "get miner hashrate",
				Flags:  rpcFlags(),
				Action: rpcAction("miner", "hashrate"),
			},
			{
				Name:   "getthreads",
				Usage:  "get miner thread number",
				Flags:  rpcFlags(),
				Action: rpcAction("miner", "getThreads"),
			},
			{
				Name:   "setthreads",
				Usage:  "set miner thread number",
				Flags:  rpcFlags(threadsFlag),
				Action: rpcAction("miner", "setThreads"),
			},
			{
				Name:   "setcoinbase",
				Usage:  "set miner coinbase",
				Flags:  rpcFlags(coinbaseFlag),
				Action: rpcAction("miner", "setCoinbase"),
			},
			{
				Name:   "getcoinbase",
				Usage:  "get miner coinbase",
				Flags:  rpcFlags(),
				Action: rpcAction("miner", "getCoinbase"),
			},
			{
				Name:   "status",
				Usage:  "get miner status",
				Flags:  rpcFlags(),
				Action: rpcAction("miner", "status"),
			},
		},
	}

	sort.Sort(cli.CommandsByName(minerCommands.Subcommands))
	sort.Sort(cli.FlagsByName(minerCommands.Flags))

	p2pCommands := cli.Command{
		Name:  "p2p",
		Usage: "p2p commands",
		Subcommands: []cli.Command{
			{
				Name:   "peers",
				Usage:  "get p2p peer connections",
				Flags:  rpcFlags(),
				Action: rpcAction("network", "getPeerCount"),
			},
			{
				Name:   "peersinfo",
				Usage:  "get p2p peers information",
				Flags:  rpcFlags(),
				Action: rpcAction("network", "getPeersInfo"),
			},
			{
				Name:   "networkversion",
				Usage:  "get current network version",
				Flags:  rpcFlags(),
				Action: rpcAction("network", "getNetworkVersion"),
			},
			{
				Name:   "protocolversion",
				Usage:  "get seele protocol version",
				Flags:  rpcFlags(),
				Action: rpcAction("network", "getProtocolVersion"),
			},
		},
	}

	sort.Sort(cli.CommandsByName(p2pCommands.Subcommands))
	sort.Sort(cli.FlagsByName(p2pCommands.Flags))

	app.Commands = []cli.Command{
		minerCommands,
		p2pCommands,
		{
			Name:   "getinfo",
			Usage:  "get node info",
			Flags:  rpcFlags(),
			Action: rpcAction("seele", "getInfo"),
		},
		{
			Name:   "getbalance",
			Usage:  "get balance info",
			Flags:  rpcFlags(accountFlag),
			Action: rpcAction("seele", "getBalance"),
		},
		{
			Name:   "sendtx",
			Usage:  "send transaction to node",
			Flags:  rpcFlags(fromFlag, toFlag, amountFlag, feeFlag, payloadFlag, nonceFlag),
			Action: rpcActionEx("seele", "addTx", makeTransaction, onTxAdded),
		},
		{
			Name:   "getnonce",
			Usage:  "get account nonce",
			Flags:  rpcFlags(accountFlag),
			Action: rpcAction("seele", "getAccountNonce"),
		},
		{
			Name:   "call",
			Usage:  "call contract",
			Flags:  rpcFlags(toFlag, payloadFlag, heightFlag),
			Action: rpcAction("seele", "call"),
		},
		{
			Name:   "getblockheight",
			Usage:  "get block height",
			Flags:  rpcFlags(),
			Action: rpcAction("seele", "getBlockHeight"),
		},
		{
			Name:   "getblock",
			Usage:  "get block by height or hash",
			Flags:  rpcFlags(hashFlag, heightFlag, fulltxFlag),
			Action: rpcAction("seele", "getBlock"),
		},
		{
			Name:   "getlogs",
			Usage:  "get logs",
			Flags:  rpcFlags(heightFlag, contractFlag, topicFlag),
			Action: rpcAction("seele", "getLogs"),
		},
		{
			Name:   "gettxpoolcontent",
			Usage:  "get transaction pool contents",
			Flags:  rpcFlags(),
			Action: rpcAction("debug", "getTxPoolContent"),
		},
		{
			Name:   "gettxpoolcount",
			Usage:  "get transaction pool transaction count",
			Flags:  rpcFlags(),
			Action: rpcAction("debug", "getTxPoolTxCount"),
		},
		{
			Name:   "getblocktxcount",
			Usage:  "get block transaction count by block height or block hash",
			Flags:  rpcFlags(hashFlag, heightFlag),
			Action: rpcAction("txpool", "getBlockTransactionCount"),
		},
		{
			Name:   "gettxinblock",
			Usage:  "get transaction by block height or block hash with index of the transaction in the block",
			Flags:  rpcFlags(hashFlag, heightFlag, indexFlag),
			Action: rpcAction("txpool", "getTransactionByBlockIndex"),
		},
		{
			Name:   "gettxbyhash",
			Usage:  "get transaction by transaction hash",
			Flags:  rpcFlags(hashFlag),
			Action: rpcAction("txpool", "getTransactionByHash"),
		},
		{
			Name:   "getdebtbyhash",
			Usage:  "get debt by debt hash",
			Flags:  rpcFlags(hashFlag),
			Action: rpcAction("txpool", "getDebtByHash"),
		},
		{
			Name:   "getdebts",
			Usage:  "get pending debts",
			Flags:  rpcFlags(),
			Action: rpcAction("privatetxpool", "getPendingDebts"),
		},
		{
			Name:   "getreceipt",
			Usage:  "get receipt by transaction hash",
			Flags:  rpcFlags(hashFlag),
			Action: rpcAction("txpool", "getReceiptByTxHash"),
		},
		{
			Name:   "getpendingtxs",
			Usage:  "get pending transactions",
			Flags:  rpcFlags(),
			Action: rpcAction("privatetxpool", "getPendingTransactions"),
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
				payloadFlag,
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
			Name:   "dumpheap",
			Usage:  "dump heap for profiling, return the file path",
			Flags:  rpcFlags(dumpFileFlag, gcBeforeDumpFlag),
			Action: rpcAction("debug", "dumpHeap"),
		},
	}

	// sort commands and flags by name
	sort.Sort(cli.CommandsByName(app.Commands))
	sort.Sort(cli.FlagsByName(app.Flags))

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
