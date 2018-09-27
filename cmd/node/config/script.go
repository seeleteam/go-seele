/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"reflect"

	"github.com/seeleteam/go-seele/cmd/node/cmd"
	"github.com/seeleteam/go-seele/cmd/util"
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

const (
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
	nodeFilePath := filepath.Join(configPath, nodeFile)
	config, err := cmd.GetConfigFromFile(nodeFilePath)
	if err != nil {
		fmt.Println("Failed to get util.Config, err:", err)
		return
	}

	hostsFilePath := filepath.Join(configPath, hostsFile)
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

	if err = changed(config, groups[ip].Host, groups[ip].Shard); err != nil {
		fmt.Println("Failed to change base info, err:", err)
		return
	}

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
	if err = json.Indent(&foutpot, output, "", "\t"); err != nil {
		fmt.Println("Failed to marshalIndet, err:", err)
		return
	}

	if err = common.SaveFile(filepath.Join(configPath, "config", nodeFile), foutpot.Bytes()); err != nil {
		fmt.Println("Failed to write file err:", err)
	}
}

// changed change the config base info
func changed(config *util.Config, host string, shard uint) error {
	config.GenesisConfig.ShardNumber = shard
	config.BasicConfig.Name = fmt.Sprint("seele_node_", host)
	config.BasicConfig.DataDir = fmt.Sprint("seele_node_", host)
	publicKey, privateKey, err := util.GenerateKey(shard)
	if err != nil {
		return err
	}

	pubkeyStr := publicKey.ToHex()
	prikeyStr := hexutil.BytesToHex(crypto.FromECDSA(privateKey))

	config.BasicConfig.Coinbase = pubkeyStr
	config.P2PConfig.SubPrivateKey = prikeyStr
	config.MetricsConfig.Addr = metricsInfo
	config.LogConfig.IsDebug = false
	config.LogConfig.PrintLog = false

	return nil
}
