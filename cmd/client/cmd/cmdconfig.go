/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"github.com/seeleteam/go-seele/seele"
)

// Request cmd request for cobra command
type Request struct {
	Use              string            // Use is the one-line usage message
	Short            string            // Short is the short description shown in the 'help' output
	Long             string            // Long is the long message shown in the 'help <this-command>' output
	ParamReflectType string            // ParamReflectType is the type of param used to visit rpc api,basic types and non nested struct is supported
	Method           string            // Method is the service method name
	UseWebsocket     bool              // UseWebsocket is how to visit the rpc api, if true will use websocket,otherwise use rpc 2.0
	Params           []*Param          // Params is the param args for cmd input line
	Handler          func(interface{}) // handler of the rpc result value
}

// Param cmd request Params for cobra command
type Param struct {
	ReflectName  string      // ReflectName is the name of property in the param  which is used to visit rpc api
	FlagName     string      // FlagName is the name of the argument which to store the value of the flag
	ShortFlag    string      // ShortFlag is the short name of the argument which to store the value of the flag,when it is "", it means not use short.
	ParamType    string      // ParamType is the type of the flag
	DefaultValue interface{} // DefaultValue is the default value of the flag when the flag is not input
	Usage        string      // Usage is the description of the flag
	Required     bool        // Required is mark the flag is required or not
}

// Config common cmd config
type Config struct {
	structMap map[string]interface{} // structMap is struct mapping for ParamReflectType in Request
	basicMap  map[string]interface{} // basicMap is basic mapping for ParamReflectType in Request
	request   []*Request             // Request is collection of cmd request for cobra command
}

// NewConfig create new Config pointer
func NewConfig() *Config {
	config := &Config{
		structMap: make(map[string]interface{}),
		basicMap:  make(map[string]interface{}),
		request:   NewCmdData(),
	}
	config.InitBasicData()
	config.InitStructData()

	return config
}

// InitStructData init struct data for cmd config
func (c *Config) InitStructData() {
	c.structMap["GetBlocksRequest"] = seele.GetBlocksRequest{}
	c.structMap["GetBlockByHashRequest"] = seele.GetBlockByHashRequest{}
	c.structMap["GetTxByBlockHeightAndIndexRequest"] = seele.GetTxByBlockHeightAndIndexRequest{}
	c.structMap["GetTxByBlockHashAndIndexRequest"] = seele.GetTxByBlockHashAndIndexRequest{}
	c.structMap["GetBlockByHashRequest"] = seele.GetBlockByHashRequest{}
	c.structMap["DumpHeapRequest"] = seele.DumpHeapRequest{}
	c.structMap["GetLogsRequest"] = seele.GetLogsRequest{}
}

// InitBasicData init basic data for cmd config
func (c *Config) InitBasicData() {
	c.basicMap["string"] = nil
	c.basicMap["int"] = nil
	c.basicMap["uint"] = nil
	c.basicMap["bool"] = nil
	c.basicMap["int64"] = nil
	c.basicMap["uint64"] = nil
	c.basicMap["nil"] = nil
}

// ParsePointInterface return parse interface, if it is a pointer, take the value
func ParsePointInterface(i interface{}) interface{} {
	switch i.(type) {
	case *string:
		return *i.(*string)
	case *bool:
		return *i.(*bool)
	case *int64:
		return *i.(*int64)
	case *int:
		return *i.(*int)
	case *uint64:
		return *i.(*uint64)
	case *uint:
		return *i.(*uint)
	default:
		return i
	}
}

// AddStructData add struct data for cmd config
func (c *Config) AddStructData(name string, value interface{}) {
	c.structMap[name] = value
}

// AddBasicData add basic data for cmd config
func (c *Config) AddBasicData(name string, value interface{}) {
	c.basicMap[name] = value
}
