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
			UseWebsocket:     true,
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
			UseWebsocket:     false,
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
			UseWebsocket:     false,
			Params:           []*Param{},
		},
	}
}
