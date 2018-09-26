package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"reflect"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log/comm"
	"github.com/seeleteam/go-seele/metrics"
	"github.com/seeleteam/go-seele/node"
)

//P2PConfig is the Configuration of p2p
type p2pConfig struct {
	// p2p.server will listen for incoming tcp connections. And it is for udp address used for Kad protocol
	ListenAddr string `json:"address"`

	// NetworkID used to define net type, for example main net and test net.
	NetworkID string `json:"networkID"`

	// static nodes which will be connected to find more nodes when the node starts
	StaticNodes []string `json:"staticNodes"`

	// SubPrivateKey which will be make PrivateKey
	SubPrivateKey string `json:"privateKey"`
}

// GenesisInfo genesis info for generating genesis block, it could be used for initializing account balance
type GenesisInfo struct {
	// Accounts accounts info for genesis block used for test
	// map key is account address -> value is account balance
	// Accounts map[common.Address]*big.Int `json:"accounts"`

	// Difficult initial difficult for mining. Use bigger difficult as you can. Because block is choose by total difficult
	Difficult int64 `json:"difficult"`

	// ShardNumber is the shard number of genesis block.
	ShardNumber uint `json:"shard"`
}

// Config is the Configuration of node
type Config struct {
	//Config is the Configuration of log
	LogConfig comm.LogConfig `json:"log"`

	// basic config for Node
	BasicConfig node.BasicConfig `json:"basic"`

	// The configuration of p2p network
	P2PConfig p2pConfig `json:"p2p"`

	// HttpServer config for http server
	HTTPServer node.HTTPServer `json:"httpServer"`

	// The configuration of websocket rpc service
	WSServerConfig node.WSServerConfig `json:"wsserver"`

	// metrics config info
	MetricsConfig *metrics.Config `json:"metrics"`

	// genesis config info
	GenesisConfig GenesisInfo `json:"genesis"`
}

// GroupInfo is hosts info of groups
type GroupInfo struct {
	Host  string `json:"host"`
	Shard uint   `json:"shard"`
	Tag   string `json:"tag"`
}

var (
	configPath  = "/home/seele/node/getconfig/"
	nodeFile    = "node.json"
	hostsFile   = "hosts.json"
	port        = "8057"
	staticNum   = 20
	metricsInfo = "0.0.0.0:8087"
	tag         = "scan"
)

func main() {
	getConfigTemp()
}

func getConfigTemp() {
	var config Config
	nodeFilePath := fmt.Sprint(configPath, nodeFile)
	buff, err := ioutil.ReadFile(nodeFilePath)
	if err != nil {
		fmt.Println("Failed to read file, filepath:", nodeFilePath, ",get config template err:", err)
		return
	}

	if err = json.Unmarshal(buff, &config); err != nil {
		fmt.Println("Failed to convert:", reflect.ValueOf(buff).String(), ",json Unmarshal err:", err)
		return
	}

	hostsFilePath := fmt.Sprint(configPath, hostsFile)
	groups := make(map[string]GroupInfo)
	buff, err = ioutil.ReadFile(hostsFilePath)
	if err != nil {
		fmt.Println("Failed to read file, filepath:", hostsFilePath, ",get hosts template err:", err)
		return
	}

	if err = json.Unmarshal(buff, &groups); err != nil {
		fmt.Println("Failed to convert:", reflect.ValueOf(buff).String(), ",json Unmarshal err:", err)
		return
	}

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println("Failed to get ip err:", err)
		return
	}

	var ip string
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
				break
			}
		}
	}

	shard := groups[ip].Shard
	config.GenesisConfig.ShardNumber = shard
	config.BasicConfig.Name = fmt.Sprint("seele_node_", groups[ip].Host)
	config.BasicConfig.DataDir = fmt.Sprint("seele_node_", groups[ip].Host)
	publicKey, privateKey := getkey(&shard)
	config.BasicConfig.Coinbase = publicKey
	config.P2PConfig.SubPrivateKey = privateKey
	config.MetricsConfig.Addr = metricsInfo
	config.LogConfig.IsDebug = false
	config.LogConfig.PrintLog = false
	count := 0
	ipList := []string{}
	for k, _ := range groups {
		if k != ip && groups[k].Tag != tag {
			host := groups[k].Host + ":" + port
			ipList = append(ipList, host)
			count++
		}

		if count > staticNum {
			break
		}
	}

	config.P2PConfig.StaticNodes = ipList
	output, err := json.Marshal(config)
	if err != nil {
		fmt.Println("Failed to convert to json err:", err)
		return
	}

	var foutpot bytes.Buffer
	json.Indent(&foutpot, output, "", "\t")

	path := fmt.Sprint(configPath, "config/")

	if err = os.MkdirAll(path, os.ModePerm); err != nil {
		fmt.Println("Failed to make dir err:", err)
		return
	}

	err = ioutil.WriteFile(fmt.Sprint(path, nodeFile), foutpot.Bytes(), os.ModePerm)
	if err != nil {
		fmt.Println("Failed to write file err:", err)
	}
}

func getkey(shard *uint) (string, string) {
	var publicKey *common.Address
	var privateKey *ecdsa.PrivateKey
	var err error
	if *shard > common.ShardCount {
		fmt.Printf("not supported shard number, shard number should be [0, %d]\n", common.ShardCount)
		return "", ""
	} else if *shard == 0 {
		publicKey, privateKey, err = crypto.GenerateKeyPair()
		if err != nil {
			fmt.Printf("Failed to generate the key pair: %s\n", err.Error())
			return "", ""
		}
	} else {
		publicKey, privateKey = crypto.MustGenerateShardKeyPair(*shard)
	}
	pubkey := publicKey.ToHex()
	prikey := hexutil.BytesToHex(crypto.FromECDSA(privateKey))
	return pubkey, prikey
}
