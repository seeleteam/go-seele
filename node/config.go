/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package node

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"crypto/ecdsa"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/p2p/discovery"
	"github.com/seeleteam/go-seele/seele"
)

// Note to add enough comments for every field
type Config struct {
	//Config is the Configuration of log
	Log log.Config `json:"log"`

	// basic config for Node
	Basic Basic `json:"basic"`

	// The configuration of p2p network
	P2P p2p.Config `json:"p2p"`

	// HttpServer config for http server
	HTTPServer HTTPServer `json:"httpServer"`

	// The SeeleConfig is the configuration to create the seele service.
	SeeleConfig seele.Config
}

// Basic config for Node
type Basic struct {
	// The name of the node
	Name string `json:"name"`

	// The version of the node
	Version string `json:"version"`

	// The file system path of the node, used to store data
	DataDir string `json:"dataDir"`

	// RPCAddr is the address on which to start RPC server.
	RPCAddr string `json:"address"`

	// coinbase used by the miner
	Coinbase string `json:"coinbase"`
		
	// capacity of the transaction pool
	Capacity uint `json:"capacity"`
}

// HTTPServer config for http server
type HTTPServer struct {
	// The HTTPAddr is the address of HTTP rpc service
	HTTPAddr string `json:"address"`

	// HTTPCors is the Cross-Origin Resource Sharing header to send to requesting
	// clients. Please be aware that CORS is a browser enforced security, it's fully
	// useless for custom HTTP clients.
	HTTPCors []string `json:"crosssorgins"`

	// HTTPHostFilter is the whitelist of hostnames which are allowed on incoming requests.
	HTTPWhiteHost []string `json:"whiteHost"`
}

// GetConfigFromFile unmarshals the config from the given file
func GetConfigFromFile(filepath string) (Config, error) {
	var config Config
	buff, err := ioutil.ReadFile(filepath)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(buff, &config)
	return config, err
}

// LoadConfigFromFile gets node config from the given file
func LoadConfigFromFile(configFile string, genesisConfigFile string) (*Config, error) {
	config, err := GetConfigFromFile(configFile)
	if err != nil {
		return nil, err
	}

	config.P2P, err = GetP2pConfig(config)
	if err != nil {
		return &config, err
	}

	if genesisConfigFile != "" {
		info, err := GetGenesisInfoFromFile(genesisConfigFile)
		if err != nil {
			return nil, err
		}
		config.SeeleConfig.GenesisConfig = info
	}

	config.SeeleConfig.Coinbase = common.HexMustToAddres(config.Basic.Coinbase)
	config.SeeleConfig.NetworkID = config.P2P.NetworkID
	config.SeeleConfig.TxConf.Capacity = config.Basic.Capacity

	common.PrintLog = config.Log.PrintLog
	common.IsDebug = config.Log.IsDebug
	config.Basic.DataDir = filepath.Join(common.GetDefaultDataFolder(), config.Basic.DataDir)
	return &config, nil
}

// GetP2pConfig ResolveStaticNodes from the given config
func GetP2pConfig(config Config) (p2p.Config, error) {

	if(config.P2P.ResolveStaticNodes != nil && config.P2P.PrivateKey != nil){
		return config.P2P,nil
	}

	if(config.P2P.ResolveStaticNodes == nil){
		resolveStaticNodes,err := GetP2pConfigResolveStaticNodes(config)
		if(err != nil){
			config.P2P.ResolveStaticNodes = resolveStaticNodes
		}
	}

	if(config.P2P.PrivateKey == nil){
		privateKey,err := GetP2pConfigPrivateKey(config)
		if(err != nil){
			config.P2P.PrivateKey = privateKey
		}
	}
	return config.P2P, nil
}

// GetP2pConfig ResolveStaticNodes from the given config
func GetP2pConfigResolveStaticNodes(config Config) ([]*discovery.Node, error) {

	if(config.P2P.ResolveStaticNodes != nil){
		return config.P2P.ResolveStaticNodes,nil
	}

	if len(config.P2P.StaticNodes) != 0  && len(config.P2P.ResolveStaticNodes) == 0 {
		for _, id := range config.P2P.StaticNodes {
			n, err := discovery.NewNodeFromString(id)
			if err != nil {
				return nil, err
			}

			config.P2P.ResolveStaticNodes = append(config.P2P.ResolveStaticNodes, n)
		}
	}

	return config.P2P.ResolveStaticNodes, nil
}

// GetP2pConfig privateKey from the given config
func GetP2pConfigPrivateKey(config Config) (*ecdsa.PrivateKey, error) {

	if(config.P2P.PrivateKey != nil){
		return config.P2P.PrivateKey,nil
	}
	key, err := crypto.LoadECDSAFromString(config.P2P.ServerPrivateKey)
	
	if err != nil {
		return nil, err
	}
	return key,err
}

// GetGenesisInfoFromFile get genesis info from a specific file
func GetGenesisInfoFromFile(filepath string) (core.GenesisInfo, error) {
	var info core.GenesisInfo
	buff, err := ioutil.ReadFile(filepath)
	if err != nil {
		return info, err
	}

	err = json.Unmarshal(buff, &info)
	return info, err
}
