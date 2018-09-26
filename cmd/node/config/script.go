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

	"github.com/seeleteam/go-seele/cmd/node/cmd"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/p2p/discovery"
)

// GroupInfo is hosts info of groups
type GroupInfo struct {
	Host  string `json:"host"`
	Shard uint   `json:"shard"`
	Tag   string `json:"tag"`
}

var (
	configPath = "/home/seele/node/getconfig/"
	nodeFile   = "node.json"
	hostsFile  = "hosts.json"
	port       = 8057
	staticNum  = 20
	// metricsInfo changed to the metric host ip when used
	metricsInfo = "0.0.0.0:8087"
	tag         = "scan"
)

func main() {
	getConfigTemp()
}

func getConfigTemp() {
	nodeFilePath := fmt.Sprint(configPath, nodeFile)
	config, err := cmd.GetConfigFromFile(nodeFilePath)
	if err != nil {
		fmt.Println("Failed to get util.Config, err:", err)
		return
	}

	hostsFilePath := fmt.Sprint(configPath, hostsFile)
	groups := make(map[string]GroupInfo)
	buff, err := ioutil.ReadFile(hostsFilePath)
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
	nodes := make([]*discovery.Node, 0)
	for k, _ := range groups {
		if k != ip && groups[k].Tag != tag {
			var node discovery.Node
			addr, err := net.ResolveIPAddr("ip", groups[k].Host)
			if err != nil {
				fmt.Println("Failed to convert host sting to ip, err:", err)
				return
			}
			node.IP = addr.IP
			node.TCPPort = port
			node.UDPPort = port
			nodes = append(nodes, &node)
			count++
		}

		if count >= staticNum {
			break
		}
	}

	config.P2PConfig.StaticNodes = nodes
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
