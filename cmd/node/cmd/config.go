/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"encoding/json"
	"io/ioutil"
	"math/big"
	"path/filepath"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/p2p/discovery"
)

// Config aggregates all configs exposed to users
// Note to add enough comments for every field
type Config struct {
	// The name of the node
	Name string

	// The version of the node
	Version string

	// The folder used to store block data
	DataDir string

	// JSON API address
	RPCAddr string

	// ServerPrivateKey private key for p2p module, do not use it as any accounts
	ServerPrivateKey string

	// network id, not used now. @TODO maybe be removed or just use Version
	NetworkID uint64

	// capacity of the transaction pool
	Capacity uint

	// coinbase used by the miner
	Coinbase string

	// static nodes which will be connected to find more nodes when the node starts
	StaticNodes []string

	// core msg interaction uses TCP address and Kademila protocol uses UDP address
	ListenAddr string

	// If IsDebug is true, the log level will be DebugLevel, otherwise it is InfoLevel
	IsDebug bool

	// If PrintLog is true, all logs will be printed in the console, otherwise they will be stored in the file.
	PrintLog bool

	// http server config info
	HttpServer HttpServer

	// websocket server config info
	WSServer WSServer
}

// GenesisInfo genesis info for generate genesis block, it could be used for initialize account balance
type GenesisInfo struct {
	// accounts info for genesis block used for test
	// map key is account address -> value is account balance
	Accounts map[string]int64
}

// HttpServer config for http server
type HttpServer struct {
	// The HTTPAddr is the address of HTTP rpc service
	HTTPAddr string

	// HTTPCors is the Cross-Origin Resource Sharing header to send to requesting
	// clients. Please be aware that CORS is a browser enforced security, it's fully
	// useless for custom HTTP clients.
	HTTPCors []string

	// HTTPHostFilter is the whitelist of hostnames which are allowed on incoming requests.
	HTTPWhiteHost []string
}

// WSServer config for websocket server
type WSServer struct {
	// The WSAddr is the address of Websocket rpc service
	WSAddr string
	// The WSAddr is the pattern of Websocket rpc service
	WSPattern string
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

// GetGenesisInfoFromFile get genesis info from a specific file
func GetGenesisInfoFromFile(filepath string) (GenesisInfo, error) {
	var info GenesisInfo
	buff, err := ioutil.ReadFile(filepath)
	if err != nil {
		return info, err
	}

	err = json.Unmarshal(buff, &info)
	return info, err
}

// GetGenesisAccountsFromFile get genesis accounts from a specific file
func GetGenesisAccountsFromFile(filepath string) (map[common.Address]*big.Int, error) {
	info, err := GetGenesisInfoFromFile(filepath)
	if err != nil {
		return nil, err
	}

	accounts := make(map[common.Address]*big.Int)
	for k, v := range info.Accounts {
		addr, err := common.HexToAddress(k)
		if err != nil {
			return nil, err
		}

		balance := big.NewInt(v)
		accounts[addr] = balance
	}

	return accounts, nil
}

// LoadConfigFromFile gets node config from the given file
func LoadConfigFromFile(configFile string, genesisConfigFile string) (*node.Config, error) {
	config, err := GetConfigFromFile(configFile)
	if err != nil {
		return nil, err
	}

	nodeConfig := new(node.Config)
	nodeConfig.Name = config.Name
	nodeConfig.Version = config.Version
	nodeConfig.RPCAddr = config.RPCAddr
	nodeConfig.HTTPAddr = config.HttpServer.HTTPAddr
	nodeConfig.HTTPCors = config.HttpServer.HTTPCors
	nodeConfig.HTTPWhiteHost = config.HttpServer.HTTPWhiteHost
	nodeConfig.WSAddr = config.WSServer.WSAddr
	nodeConfig.WSPattern = config.WSServer.WSPattern

	nodeConfig.P2P, err = GetP2pConfig(config)
	if err != nil {
		return nil, err
	}

	if genesisConfigFile != "" {
		accounts, err := GetGenesisAccountsFromFile(genesisConfigFile)
		if err != nil {
			return nil, err
		}
		nodeConfig.SeeleConfig.GenesisAccounts = accounts
	}

	nodeConfig.SeeleConfig.Coinbase = common.HexMustToAddres(config.Coinbase)
	nodeConfig.SeeleConfig.NetworkID = config.NetworkID
	nodeConfig.SeeleConfig.TxConf.Capacity = config.Capacity

	common.PrintLog = config.PrintLog
	common.IsDebug = config.IsDebug
	nodeConfig.DataDir = filepath.Join(common.GetDefaultDataFolder(), config.DataDir)
	return nodeConfig, nil
}

// GetP2pConfig gets p2p module config from the given config
func GetP2pConfig(config Config) (p2p.Config, error) {
	p2pConfig := p2p.Config{}

	if len(config.StaticNodes) != 0 {
		for _, id := range config.StaticNodes {
			n, err := discovery.NewNodeFromString(id)
			if err != nil {
				return p2p.Config{}, err
			}

			p2pConfig.StaticNodes = append(p2pConfig.StaticNodes, n)
		}
	}

	key, err := crypto.LoadECDSAFromString(config.ServerPrivateKey)
	if err != nil {
		return p2pConfig, err
	}

	p2pConfig.PrivateKey = key
	p2pConfig.ListenAddr = config.ListenAddr
	return p2pConfig, nil
}
