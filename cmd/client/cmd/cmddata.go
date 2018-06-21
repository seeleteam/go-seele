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
			Params: []*Param{
				&Param{
					ReflectName:  "HashHex",
					FlagName:     "hash",
					ShortFlag:    "",
					ParamType:    "*string",
					DefaultValue: "",
					Usage:        "hash for the block",
					Required:     true,
				},
				&Param{
					ReflectName:  "FullTx",
					FlagName:     "fulltx",
					ShortFlag:    "f",
					ParamType:    "*bool",
					DefaultValue: false,
					Usage:        "whether get full transaction info, default is false",
					Required:     false,
				},
			},
		},
		&Request{
			Use:   "getblockbyheight",
			Short: "get block info by block height",
			Long: `For example:
  			client.exe getblockbyheight --height -1 [-f=true] [-a 127.0.0.1:55027]`,
			ParamReflectType: "GetBlockByHeightRequest",
			Method:           "seele.GetBlockByHeight",
			UseWebsocket:     false,
			Params: []*Param{
				&Param{
					ReflectName:  "Height",
					FlagName:     "height",
					ShortFlag:    "",
					ParamType:    "*int64",
					DefaultValue: -1, // -1 represent the current block
					Usage:        "height for the block",
					Required:     true,
				},
				&Param{
					ReflectName:  "FullTx",
					FlagName:     "fulltx",
					ShortFlag:    "f",
					ParamType:    "*bool",
					DefaultValue: false,
					Usage:        "whether get full transaction info, default is false",
					Required:     false,
				},
			},
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
			Params: []*Param{
				&Param{
					ReflectName:  "Height",
					FlagName:     "height",
					ShortFlag:    "",
					ParamType:    "*int64",
					DefaultValue: -1,
					Usage:        "height for the block",
					Required:     true,
				},
			},
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
			Params: []*Param{
				&Param{
					ReflectName:  "Height",
					FlagName:     "height",
					ShortFlag:    "",
					ParamType:    "*int64",
					DefaultValue: -1, // -1 represent the current block
					Usage:        "height for get block transaction count",
					Required:     false,
				},
			},
		},
		&Request{
			Use:   "getblocktxcountbyhash",
			Short: "get block transaction count by hash",
			Long: `For example:
  			client.exe getblocktxcountbyhash --hash 0x00000006f1c704b54ba9c7d9a3d50982d0479680afcf62d3e69bc42b30e595fd`,
			ParamReflectType: "string",
			Method:           "txpool.GetBlockTransactionCountByHash",
			Params: []*Param{
				&Param{
					ReflectName:  "HashHex",
					FlagName:     "hash",
					ShortFlag:    "",
					ParamType:    "*string",
					DefaultValue: "",
					Usage:        "hash for get block transaction count",
					Required:     true,
				},
			},
		},
		&Request{
			Use:   "gettxbyheightandindex",
			Short: "get transaction by block height and index",
			Long: `For example:
  			client.exe gettxbyheightandindex --height -1 --index 0`,
			ParamReflectType: "GetTxByBlockHeightAndIndexRequest",
			Method:           "txpool.GetTransactionByBlockHeightAndIndex",
			Params: []*Param{
				&Param{
					ReflectName:  "Height",
					FlagName:     "height",
					ShortFlag:    "",
					ParamType:    "*int64",
					DefaultValue: -1,
					Usage:        "height for get block",
					Required:     false,
				},
				&Param{
					ReflectName:  "Index",
					FlagName:     "index",
					ShortFlag:    "",
					ParamType:    "*int",
					DefaultValue: 0,
					Usage:        "index of the transaction in block",
					Required:     false,
				},
			},
		},
		&Request{
			Use:   "gettxbyhashandindex",
			Short: "get transaction by hash and index",
			Long: `For example:
  			client.exe gettxbyhashandindex --hash 0x00000006f1c704b54ba9c7d9a3d50982d0479680afcf62d3e69bc42b30e595fd --index 0`,
			ParamReflectType: "GetTxByBlockHashAndIndexRequest",
			Method:           "txpool.GetTransactionByBlockHashAndIndex",
			Params: []*Param{
				&Param{
					ReflectName:  "HashHex",
					FlagName:     "hash",
					ShortFlag:    "",
					ParamType:    "*string",
					DefaultValue: "",
					Usage:        "hash for get block",
					Required:     true,
				},
				&Param{
					ReflectName:  "Index",
					FlagName:     "index",
					ShortFlag:    "",
					ParamType:    "*int",
					DefaultValue: 0,
					Usage:        "index of the transaction in block",
					Required:     false,
				},
			},
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
			Use:   "gettransactionbyhash",
			Short: "get transaction info by hash",
			Long: `For example:
  			client.exe gettransactionbyhash --hash 0xf5aa155ae1d0a126195a70bda69c7f1db0a728f7f860f33244fee83703a80195`,
			ParamReflectType: "string",
			Method:           "txpool.GetTransactionByHash",
			UseWebsocket:     false,
			Params: []*Param{
				&Param{
					ReflectName:  "TxHash",
					FlagName:     "hash",
					ShortFlag:    "",
					ParamType:    "*string",
					DefaultValue: "",
					Usage:        "hash of the transaction",
					Required:     true,
				},
			},
		},
		&Request{
			Use:   "getreceiptbytxhash",
			Short: "get receipt by tx hash",
			Long: `For example:
  			client.exe getreceiptbytxhash --hash 0xf5aa155ae1d0a126195a70bda69c7f1db0a728f7f860f33244fee83703a80195`,
			ParamReflectType: "string",
			Method:           "txpool.GetReceiptByTxHash",
			UseWebsocket:     false,
			Params: []*Param{
				&Param{
					ReflectName:  "TxHash",
					FlagName:     "hash",
					ShortFlag:    "",
					ParamType:    "*string",
					DefaultValue: "",
					Usage:        "hash of the transaction",
					Required:     true,
				},
			},
		},
		&Request{
			Use:   "setminerthreads",
			Short: "set miner threads",
			Long: `For example:
  			client.exe setminerthreads -t 2`,
			ParamReflectType: "nil",
			Method:           "miner.SetThreads",
			UseWebsocket:     false,
			Params: []*Param{
				&Param{
					ReflectName:  "thread",
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
			ParamReflectType: "nil",
			Method:           "miner.SetCoinbase",
			UseWebsocket:     false,
			Params: []*Param{
				&Param{
					ReflectName:  "coinbaseStr",
					FlagName:     "coinbase",
					ShortFlag:    "c",
					ParamType:    "*string",
					DefaultValue: "",
					Usage:        "coinbase of the miner",
					Required:     true,
				},
			},
			Handler: func(interface{}) { fmt.Println("miner set coinbase succeed") },
		},
	}
}
