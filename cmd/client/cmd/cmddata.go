/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"

	"github.com/seeleteam/go-seele/common/hexutil"
)

// NewCmdData load all cmd data for init
func NewCmdData() []*Request {
	return []*Request{
		&Request{
			Use:   "getblockbyhash",
			Short: "get block info by block hash",
			Long: `For example:
  			client.exe getblockbyhash --hash 0x0000009721cf7bb5859f1a0ced952fcf71929ff8382db6ef20041ed441d5f92f [-f=true] [-a 127.0.0.1:55027]`,
			ParamReflectType: "GetBlockByHashRequest",
			Method:           "seele.GetBlockByHash",
			UseWebsocket:     false,
			Params:           []*Param{paramBlockHash, paramFullTx},
		},
		&Request{
			Use:   "getblockbyheight",
			Short: "get block info by block height",
			Long: `For example:
  			client.exe getblockbyheight --height -1 [-f=true] [-a 127.0.0.1:55027]`,
			ParamReflectType: "GetBlockByHeightRequest",
			Method:           "seele.GetBlockByHeight",
			UseWebsocket:     false,
			Params:           []*Param{paramBlockHeight, paramFullTx},
			Handler: func(v interface{}) {
				result := v.(map[string]interface{})
				txs := result["transactions"].([]interface{})
				fmt.Printf("transaction numbers: %d\n", len(txs))
			},
		},
		&Request{
			Use:   "getblockrlp",
			Short: "get block rlp hex by block height",
			Long: `For example:
  			client.exe getblockrlp --height -1 [-a 127.0.0.1:55027]`,
			ParamReflectType: "int64",
			Method:           "debug.GetBlockRlp",
			UseWebsocket:     false,
			Params:           []*Param{paramBlockHeight},
			Handler: func(i interface{}) {
				v := i.(string)
				buff, err := hexutil.HexToBytes(v)
				if err != nil {
					fmt.Println("hex to byte failed ", err)
					return
				}

				fmt.Printf("block size: %d byte", len(buff))
			},
		},
		&Request{
			Use:   "getblockheight",
			Short: "get block height of the chain head",
			Long: `For example:
  			client.exe getblockheight`,
			ParamReflectType: "nil",
			Method:           "seele.GetBlockHeight",
			UseWebsocket:     false,
			Params:           []*Param{},
		},
		&Request{
			Use:   "getblocktxcountbyheight",
			Short: "get block transaction count by height",
			Long: `For example:
  			client.exe getblocktxcountbyheight --height -1`,
			ParamReflectType: "int64",
			Method:           "txpool.GetBlockTransactionCountByHeight",
			Params:           []*Param{paramBlockHeight},
		},
		&Request{
			Use:   "getblocktxcountbyhash",
			Short: "get block transaction count by hash",
			Long: `For example:
  			client.exe getblocktxcountbyhash --hash 0x00000006f1c704b54ba9c7d9a3d50982d0479680afcf62d3e69bc42b30e595fd`,
			ParamReflectType: "string",
			Method:           "txpool.GetBlockTransactionCountByHash",
			Params:           []*Param{paramBlockHash},
		},
		&Request{
			Use:   "gettxbyheightandindex",
			Short: "get transaction by block height and index",
			Long: `For example:
  			client.exe gettxbyheightandindex --height -1 --index 0`,
			ParamReflectType: "GetTxByBlockHeightAndIndexRequest",
			Method:           "txpool.GetTransactionByBlockHeightAndIndex",
			Params:           []*Param{paramBlockHeight, paramTxIndex},
		},
		&Request{
			Use:   "gettxbyhashandindex",
			Short: "get transaction by hash and index",
			Long: `For example:
  			client.exe gettxbyhashandindex --hash 0x00000006f1c704b54ba9c7d9a3d50982d0479680afcf62d3e69bc42b30e595fd --index 0`,
			ParamReflectType: "GetTxByBlockHashAndIndexRequest",
			Method:           "txpool.GetTransactionByBlockHashAndIndex",
			Params:           []*Param{paramBlockHash, paramTxIndex},
		},
		&Request{
			Use:   "getpeercount",
			Short: "get count of connected peers",
			Long: `For example:
	  		client.exe getpeercount [-a 127.0.0.1:55027]`,
			ParamReflectType: "nil",
			Method:           "network.GetPeerCount",
			UseWebsocket:     false,
			Params:           []*Param{},
		},
		&Request{
			Use:   "getnetworkversion",
			Short: "get current network version",
			Long: `For example:
	  		client.exe getnetworkversion [-a 127.0.0.1:55027]`,
			ParamReflectType: "nil",
			Method:           "network.GetNetworkVersion",
			UseWebsocket:     false,
			Params:           []*Param{},
		},
		&Request{
			Use:   "getprotocolversion",
			Short: "get seele protocol version",
			Long: `For example:
	  		client.exe getprotocolversion [-a 127.0.0.1:55027]`,
			ParamReflectType: "nil",
			Method:           "network.GetProtocolVersion",
			UseWebsocket:     false,
			Params:           []*Param{},
		},
		&Request{
			Use:   "getpeersinfo",
			Short: "get seele peers info",
			Long: `For example:
	  		client.exe getpeersinfo [-a 127.0.0.1:55027]`,
			ParamReflectType: "nil",
			Method:           "network.GetPeersInfo",
			UseWebsocket:     false,
			Params:           []*Param{},
		},
		&Request{
			Use:   "gettxbyhash",
			Short: "get transaction info by hash",
			Long: `For example:
  			client.exe gettxbyhash --hash 0xf5aa155ae1d0a126195a70bda69c7f1db0a728f7f860f33244fee83703a80195`,
			ParamReflectType: "string",
			Method:           "txpool.GetTransactionByHash",
			UseWebsocket:     false,
			Params:           []*Param{paramTxHash},
		},
		&Request{
			Use:   "getreceiptbytxhash",
			Short: "get receipt by tx hash",
			Long: `For example:
  			client.exe getreceiptbytxhash --hash 0xf5aa155ae1d0a126195a70bda69c7f1db0a728f7f860f33244fee83703a80195`,
			ParamReflectType: "string",
			Method:           "txpool.GetReceiptByTxHash",
			UseWebsocket:     false,
			Params:           []*Param{paramTxHash},
		},
		&Request{
			Use:   "setminerthreads",
			Short: "set miner threads",
			Long: `For example:
  			client.exe setminerthreads -t 2`,
			ParamReflectType: "int",
			Method:           "miner.SetThreads",
			UseWebsocket:     false,
			Params: []*Param{
				&Param{
					ReflectName:  "Thread",
					FlagName:     "thread",
					ShortFlag:    "t",
					ParamType:    "*int",
					DefaultValue: 0,
					Usage:        "threads of the miner",
					Required:     true,
				},
			},
			Handler: func(interface{}) { fmt.Println("succeed to set miner thread number") },
		},
		&Request{
			Use:   "setcoinbase",
			Short: "set coinbase",
			Long: `For example:
  			client.exe setcoinbase -c "0x4c10f2cd2159bb432094e3be7e17904c2b4aeb21"`,
			ParamReflectType: "string",
			Method:           "miner.SetCoinbase",
			UseWebsocket:     false,
			Params: []*Param{
				&Param{
					ReflectName:  "Coinbase",
					FlagName:     "coinbase",
					ShortFlag:    "c",
					ParamType:    "*string",
					DefaultValue: "",
					Usage:        "coinbase of the miner",
					Required:     true,
				},
			},
			Handler: func(interface{}) { fmt.Println("succeed to set miner coinbase") },
		},
		&Request{
			Use:              "getdownloadstatus",
			Short:            "get the download status of block synchronization",
			Long:             "Get the download status of block synchronization",
			ParamReflectType: "nil",
			Method:           "download.GetStatus",
			UseWebsocket:     false,
			Params:           []*Param{},
		},
		&Request{
			Use:   "getminerthreads",
			Short: "get miner threads",
			Long: `For example:
  			client.exe getminerthreads`,
			ParamReflectType: "nil",
			Method:           "miner.GetThreads",
			UseWebsocket:     false,
			Params:           []*Param{},
		},
		&Request{
			Use:   "getcoinbase",
			Short: "get coinbase",
			Long: `For example:
  			client.exe getcoinbase`,
			ParamReflectType: "nil",
			Method:           "miner.GetCoinbase",
			UseWebsocket:     false,
			Params:           []*Param{},
		},
		&Request{
			Use:   "getinfo",
			Short: "get the miner info",
			Long: `get the miner info
			For example:
				client.exe getinfo -a 127.0.0.1:55027`,
			ParamReflectType: "nil",
			Method:           "seele.GetInfo",
			UseWebsocket:     false,
			Params:           []*Param{},
		},
		&Request{
			Use:   "gettxpooltxcount",
			Short: "get the number of all processable transactions contained within the transaction pool",
			Long: `For example:
				client.exe gettxpooltxcount`,
			ParamReflectType: "nil",
			Method:           "debug.GetTxPoolTxCount",
			UseWebsocket:     false,
			Params:           []*Param{},
		},
		&Request{
			Use:   "printblock",
			Short: "get block pretty printed form by block height",
			Long: `For example:
				client.exe printblock --height -1 [-a 127.0.0.1:55027]`,
			ParamReflectType: "int64",
			Method:           "debug.PrintBlock",
			UseWebsocket:     false,
			Params:           []*Param{paramBlockHeight},
		},
		&Request{
			Use:              "dumpheap",
			Short:            "dump heap",
			Long:             "dump heap for profiling",
			ParamReflectType: "string",
			Method:           "debug.DumpHeap",
			UseWebsocket:     false,
			Params: []*Param{
				&Param{
					ReflectName:  "Filename",
					FlagName:     "filename",
					ShortFlag:    "f",
					ParamType:    "*string",
					DefaultValue: "",
					Usage:        "heap dump file name",
					Required:     false,
				},
			},
		},
	}
}
