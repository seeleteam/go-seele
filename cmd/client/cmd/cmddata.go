/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

// NewCmdData load all cmd data for init
func NewCmdData() []*Request {
	return []*Request{
		&Request{
			Use:              "teststruct",
			Short:            "test",
			Long:             "test",
			ParamReflectType: "GetBlockByHeightRequest",
			Method:           "seele.GetBlockByHeight",
			Params: []*Param{
				&Param{
					ReflectName:  "Height",
					ParamName:    "height",
					ShortHand:    "",
					ParamType:    "*int64",
					DefaultValue: -1,
					Usage:        "height for test",
					Required:     true,
				},
				&Param{
					ReflectName:  "FullTx",
					ParamName:    "fulltx",
					ShortHand:    "f",
					ParamType:    "*bool",
					DefaultValue: false,
					Usage:        "fulltx for test",
					Required:     false,
				},
			},
		},
		&Request{
			Use:              "testbasic",
			Short:            "test",
			Long:             "test",
			ParamReflectType: "int64",
			Method:           "debug.GetBlockRlp",
			Params: []*Param{
				&Param{
					ReflectName:  "Height",
					ParamName:    "height",
					ShortHand:    "",
					ParamType:    "*int64",
					DefaultValue: -1,
					Usage:        "height for test",
					Required:     true,
				},
			}},
		&Request{
			Use:              "testnil",
			Short:            "test",
			Long:             "test",
			ParamReflectType: "nil",
			Method:           "seele.GetBlockHeight",
			Params:           []*Param{},
		},
		&Request{
			Use:   "getblocktxcountbyheight",
			Short: "get block transaction count by height",
			Long: `For example:
  client.exe getblocktxcountbyheight --height -1`,
			ParamReflectType: "GetBlockTxCountByHeightRequest",
			Method:           "txpool.GetBlockTransactionCountByHeight",
			Params: []*Param{
				&Param{
					ReflectName:  "Height",
					ParamName:    "height",
					ShortHand:    "",
					ParamType:    "*int64",
					DefaultValue: -1,
					Usage:        "height for get block transaction count",
					Required:     false,
				},
			}},
		&Request{
			Use:   "getblocktxcountbyhash",
			Short: "get block transaction count by hash",
			Long: `For example:
  client.exe getblocktxcountbyhash --hash 0x00000006f1c704b54ba9c7d9a3d50982d0479680afcf62d3e69bc42b30e595fd`,
			ParamReflectType: "GetBlockTxCountByHashRequest",
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
	}
}
