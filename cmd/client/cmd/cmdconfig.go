/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"encoding/json"
	"io/ioutil"

	"github.com/seeleteam/go-seele/seele"
)

// cmd config paths
const (
	DefaultPath = "../cmd/client/config/cmd.json"
)

// Request cmd request for cobra command
type Request struct {
	Use         string  // Use is the one-line usage message
	Short       string  // Short is the short description shown in the 'help' output
	Long        string  // Long is the long message shown in the 'help <this-command>' output
	RequestType string  // RequestType is the type of param used to visit rpc api
	Method      string  // Method is the service method name
	Params      []Param // Params is the param args for cmd input line
}

// Param cmd request Params for cobra command
type Param struct {
	ReflectName  string      // ReflectName is the name of property in the param  which is used to visit rpc api
	ParamName    string      // ParamName is the name of the argument which to store the value of the flag
	ShortHand    string      // ShortHand is the short name of the argument which to store the value of the flag
	ParamType    string      // ParamType is the type of the flag
	DefaultValue interface{} // DefaultValue is the default value of the flag when the flag is not input
	Usage        string      // Usage is the description of the flag
	Required     bool        // Required is mark the flag is required or not
	UseShort     bool        // UseShort is need to use ShortHand or not
}

// Config common cmd config
type Config struct {
	structMap map[string]interface{}
	basicMap  map[string]interface{}
	request   []Request
}

// NewConfig create new Config pointer
func NewConfig(filepath string) (*Config, error) {
	config := &Config{
		structMap: make(map[string]interface{}),
		basicMap:  make(map[string]interface{}),
	}
	config.InitBasicData()
	config.InitStructData()
	requests, err := config.GetRequestsFromFile(filepath)
	if err != nil {
		return nil, err
	}
	config.request = requests
	return config, nil
}

// InitStructData init struct data for cmd config
func (c *Config) InitStructData() {
	c.structMap["GetBlockByHeightRequest"] = seele.GetBlockByHeightRequest{}
	c.structMap["GetBlockByHashRequest"] = seele.GetBlockByHashRequest{}
}

// InitBasicData init basic data for cmd config
func (c *Config) InitBasicData() {
	c.basicMap["string"] = nil
	c.basicMap["int"] = nil
	c.basicMap["bool"] = nil
	c.basicMap["int64"] = nil
	c.basicMap["uint64"] = nil
	c.basicMap["nil"] = nil
}

// GetRequestsFromFile get cmd request from json file
func (c *Config) GetRequestsFromFile(filePath string) ([]Request, error) {
	buff, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var requests []Request
	err = json.Unmarshal(buff, &requests)
	if err != nil {
		return nil, err
	}
	return requests, nil
}

// ParsePointInterface return parse interface,if it is a pointer, take the value
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
