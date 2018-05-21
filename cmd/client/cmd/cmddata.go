/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

// NewCmdData load all cmd data for init
func NewCmdData() []*Request {
	return []*Request{
		&Request{
			Use:   "getblockbyheight",
			Short: "get block info by block height",
			Long: `For example:
  client.exe getblockbyheight --height -1 [-f=true] [-a 127.0.0.1:55027]`,
			ParamReflectType: "GetBlockByHeightRequest",
			Method:           "seele.GetBlockByHeight",
			Params: []*Param{
				&Param{
					ReflectName:  "Height",
					ParamName:    "height",
					ShortHand:    "",
					ParamType:    "*int64",
					DefaultValue: -1, // -1 represent the current block
					Usage:        "height for the block",
					Required:     true,
				},
				&Param{
					ReflectName:  "FullTx",
					ParamName:    "fulltx",
					ShortHand:    "f",
					ParamType:    "*bool",
					DefaultValue: false,
					Usage:        "is add full tx, default is false",
					Required:     false,
				},
			},
		},
		&Request{
			Use:   "getblockrlp",
			Short: "get block rlp hex by block height",
			Long: `For example:
  client.exe getblockrlp --height -1 [-a 127.0.0.1:55027]`,
			ParamReflectType: "int64",
			Method:           "debug.GetBlockRlp",
			Params: []*Param{
				&Param{
					ReflectName:  "Height",
					ParamName:    "height",
					ShortHand:    "",
					ParamType:    "*int64",
					DefaultValue: -1,
					Usage:        "height for the block",
					Required:     true,
				},
			}},
		&Request{
			Use:   "getblockheight",
			Short: "get block height of the chain head",
			Long: `For example:
  client.exe getblockheight`,
			ParamReflectType: "nil",
			Method:           "seele.GetBlockHeight",
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
					ParamName:    "height",
					ShortHand:    "",
					ParamType:    "*int64",
					DefaultValue: -1, // -1 represent the current block
					Usage:        "height for get block transaction count",
					Required:     false,
				},
			}},
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
					ParamName:    "hash",
					ShortHand:    "",
					ParamType:    "*string",
					DefaultValue: "",
					Usage:        "hash for get block transaction count",
					Required:     true,
				},
			}},
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
					ParamName:    "height",
					ShortHand:    "",
					ParamType:    "*int64",
					DefaultValue: -1,
					Usage:        "height for get block",
					Required:     false,
				},
				&Param{
					ReflectName:  "Index",
					ParamName:    "index",
					ShortHand:    "",
					ParamType:    "*int",
					DefaultValue: 0,
					Usage:        "index of the transaction in block",
					Required:     false,
				},
			}},
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
					ParamName:    "hash",
					ShortHand:    "",
					ParamType:    "*string",
					DefaultValue: "",
					Usage:        "hash for get block",
					Required:     true,
				},
				&Param{
					ReflectName:  "Index",
					ParamName:    "index",
					ShortHand:    "",
					ParamType:    "*int",
					DefaultValue: 0,
					Usage:        "index of the transaction in block",
					Required:     false,
				},
			}},
	}
}
