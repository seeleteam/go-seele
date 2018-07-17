/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

// Common command parameters
var (
	paramBlockHash = &Param{
		ReflectName:  "HashHex",
		FlagName:     "hash",
		ShortFlag:    "",
		ParamType:    "*string",
		DefaultValue: "",
		Usage:        "block hash in HEX format",
		Required:     true,
	}

	paramFullTx = &Param{
		ReflectName:  "FullTx",
		FlagName:     "fulltx",
		ShortFlag:    "f",
		ParamType:    "*bool",
		DefaultValue: false,
		Usage:        "whether get full transaction info, default is false",
		Required:     false,
	}

	paramBlockHeight = &Param{
		ReflectName:  "Height",
		FlagName:     "height",
		ShortFlag:    "",
		ParamType:    "*int64",
		DefaultValue: -1, // negative value represents the current block
		Usage:        "block height",
		Required:     false,
	}

	paramContractAddress = &Param{
		ReflectName:  "ContractAddress",
		FlagName:     "address",
		ShortFlag:    "",
		ParamType:    "*string",
		DefaultValue: "",
		Usage:        "the contract address",
		Required:     true,
	}

	paramTopic = &Param{
		ReflectName:  "Topics",
		FlagName:     "topic",
		ShortFlag:    "",
		ParamType:    "*string",
		DefaultValue: "",
		Usage:        "event name hash",
		Required:     true,
	}

	paramTxIndex = &Param{
		ReflectName:  "Index",
		FlagName:     "index",
		ShortFlag:    "i",
		ParamType:    "*uint",
		DefaultValue: 0,
		Usage:        "index of the transaction in block",
		Required:     false,
	}

	paramTxHash = &Param{
		ReflectName:  "TxHash",
		FlagName:     "hash",
		ShortFlag:    "",
		ParamType:    "*string",
		DefaultValue: "",
		Usage:        "transaction hash in HEX format",
		Required:     true,
	}
)
