/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"path/filepath"
	"strings"
	"time"

	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/contract/system"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log/comm"
	"github.com/seeleteam/go-seele/metrics"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/p2p/discovery"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/urfave/cli"
)

var (
	errInvalidVersion        = errors.New("invalid subchain version")
	errInvalidTokenFullName  = errors.New("invalid subchain token full name")
	errInvalidTokenShortName = errors.New("invalid subchain token short name")
	errInvalidTokenAmount    = errors.New("invalid subchain token amount")
	errSubChainInfo          = errors.New("failed to get sub-chain information")

	defaultTokenFullName  = "seelecoin"
	defaultTokenShortName = "seele"
)

func registerSubChain(client *rpc.Client) (interface{}, interface{}, error) {
	amountValue = "0"

	subChain, err := getSubChainFromFile(subChainJSONFileVale)
	if err != nil {
		return nil, nil, err
	}

	if err := system.ValidateDomainName([]byte(subChain.Name)); err != nil {
		return nil, nil, err
	}

	if len(subChain.Version) == 0 {
		return nil, nil, errInvalidVersion
	}

	if len(subChain.TokenFullName) == 0 || strings.ToLower(subChain.TokenFullName) == defaultTokenFullName {
		return nil, nil, errInvalidTokenFullName
	}

	if len(subChain.TokenShortName) == 0 || strings.ToLower(subChain.TokenShortName) == defaultTokenShortName {
		return nil, nil, errInvalidTokenShortName
	}

	if subChain.TokenAmount == 0 {
		return nil, nil, errInvalidTokenAmount
	}

	subChainBytes, err := json.Marshal(subChain)
	if err != nil {
		return nil, nil, err
	}

	tx, err := sendSystemContractTx(client, system.SubChainContractAddress, system.CmdSubChainRegister, subChainBytes)
	if err != nil {
		return nil, nil, err
	}

	output := make(map[string]interface{})
	output["Tx"] = *tx
	output["SubChainName"] = subChain.Name
	output["TokenFullName"] = subChain.TokenFullName
	output["TokenShortName"] = subChain.TokenShortName

	return output, tx, err
}

func querySubChain(client *rpc.Client) (interface{}, interface{}, error) {
	amountValue = "0"

	if err := system.ValidateDomainName([]byte(nameValue)); err != nil {
		return nil, nil, err
	}

	tx, err := sendSystemContractTx(client, system.SubChainContractAddress, system.CmdSubChainQuery, []byte(nameValue))
	if err != nil {
		return nil, nil, err
	}

	return tx, tx, err
}

func createSubChainConfigFile(c *cli.Context) error {
	client, err := rpc.DialTCP(context.Background(), addressValue)
	if err != nil {
		return err
	}

	subChainInfo, err := getSubChainFromReceipt(client)
	if err != nil {
		return err
	}

	networkID, err := util.GetNetworkID(client)
	if err != nil {
		return err
	}

	config, err := getConfigFromSubChain(networkID, subChainInfo)
	if err != nil {
		return err
	}

	// save accounts file
	byteAccounts, err := json.MarshalIndent(subChainInfo.GenesisAccounts, "", "\t")
	if err != nil {
		return err
	}
	if err = common.SaveFile(filepath.Join(outPutValue, "accounts.json"), byteAccounts); err != nil {
		return err
	}

	// save node file
	byteNode, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return err
	}
	if err = common.SaveFile(filepath.Join(outPutValue, "node.json"), byteNode); err != nil {
		return err
	}

	fmt.Println("generate sub chain config files successfully")
	return nil
}

func getSubChainFromFile(filepath string) (*system.SubChainInfo, error) {
	var subChain system.SubChainInfo
	buff, err := ioutil.ReadFile(filepath)
	if err != nil {
		return &subChain, err
	}

	err = json.Unmarshal(buff, &subChain)
	return &subChain, err
}

func getSubChainFromReceipt(client *rpc.Client) (*system.SubChainInfo, error) {
	if err := system.ValidateDomainName([]byte(nameValue)); err != nil {
		return nil, err
	}
	payloadBytes := append([]byte{system.CmdSubChainQuery}, []byte(nameValue)...)
	mapReceipt, err := util.CallContract(client, system.SubChainContractAddress.Hex(), hexutil.BytesToHex(payloadBytes), -1)
	if err != nil {
		return nil, err
	}

	resultFlag, ok := mapReceipt["failed"].(bool)
	if !ok {
		return nil, errSubChainInfo
	}
	result, ok := mapReceipt["result"].(string)
	if !ok {
		return nil, errSubChainInfo
	}
	if resultFlag {
		return nil, fmt.Errorf("failed to get sub-chain information, %s", result)
	}

	bytesSubChainInfo, err := hexutil.HexToBytes(result)
	if err != nil {
		return nil, err
	}

	if len(bytesSubChainInfo) == 0 {
		return nil, fmt.Errorf("sub-chain %s does not exist", nameValue)
	}

	var subChainInfo system.SubChainInfo
	if err = json.Unmarshal(bytesSubChainInfo, &subChainInfo); err != nil {
		return nil, err
	}

	return &subChainInfo, nil
}

// getPrivateKey get private key and validate shard
func getPrivateKey() (*ecdsa.PrivateKey, error) {
	var privateKey *ecdsa.PrivateKey
	var err error
	if len(privateKeyValue) == 0 {
		_, privateKey, err = util.GenerateKey(shardValue)
		if err != nil {
			return nil, err
		}
	} else {
		privateKey, err = crypto.LoadECDSAFromString(privateKeyValue)
		if err != nil {
			return nil, fmt.Errorf("invalid key: %s", err)
		}
	}
	return privateKey, nil
}

func getStaticNodes() ([]*discovery.Node, error) {
	var arrayNode []*discovery.Node
	for _, strNode := range staticNodesValue.Value() {
		node, err := discovery.NewNodeFromIP(strNode)
		if err != nil {
			return nil, err
		}

		arrayNode = append(arrayNode, node)
	}
	return arrayNode, nil
}

func getConfigFromSubChain(networkID string, subChainInfo *system.SubChainInfo) (*util.Config, error) {
	addr, err := common.HexToAddress(coinbaseValue)
	if err != nil {
		return nil, fmt.Errorf("invalid coinbase, err:%s", err.Error())
	}
	shardNumber := addr.Shard()

	if shardValue == 0 {
		shardValue = shardNumber
	}

	if shardValue != shardNumber {
		return nil, fmt.Errorf("input shard(%d) is not equal to shard nubmer(%d) obtained from the input coinbase:%s", shardValue, shardNumber, coinbaseValue)
	}

	privateKey, err := getPrivateKey()
	if err != nil {
		return nil, err
	}

	staticNodes, err := getStaticNodes()
	if err != nil {
		return nil, err
	}

	config := &util.Config{}
	config.BasicConfig = node.BasicConfig{
		Name:           subChainInfo.Name,
		Version:        subChainInfo.Version,
		DataDir:        subChainInfo.Name,
		RPCAddr:        "0.0.0.0:8127",
		Coinbase:       coinbaseValue,
		MinerAlgorithm: algorithmValue,
	}

	config.P2PConfig = p2p.Config{
		NetworkID:     fmt.Sprintf("%s.%d.%s", subChainInfo.Name, subChainInfo.Owner.Shard(), networkID),
		ListenAddr:    "0.0.0.0:8157",
		StaticNodes:   append(subChainInfo.StaticNodes, staticNodes...),
		SubPrivateKey: hexutil.BytesToHex(crypto.FromECDSA(privateKey)),
	}

	config.LogConfig = comm.LogConfig{
		PrintLog: true,
	}

	config.HTTPServer = node.HTTPServer{
		HTTPAddr:      "127.0.0.1:8136",
		HTTPCors:      []string{"*"},
		HTTPWhiteHost: []string{"*"},
	}

	config.WSServerConfig = node.WSServerConfig{
		Address:      "127.0.0.1:8146",
		CrossOrigins: []string{"*"},
	}

	config.MetricsConfig = &metrics.Config{
		Addr:     "127.0.0.1:8187",
		Duration: time.Duration(10),
		Database: "influxdb",
		Username: "test",
		Password: "test123",
	}

	config.GenesisConfig = core.GenesisInfo{
		Difficult:       int64(subChainInfo.GenesisDifficulty),
		ShardNumber:     shardValue,
		CreateTimestamp: subChainInfo.CreateTimestamp,
	}

	return config, nil
}

func generateTemplate(c *cli.Context) error {
	if err := system.ValidateDomainName([]byte(nameValue)); err != nil {
		return err
	}

	subChainInfo := system.SubChainInfo{
		Name:              nameValue,
		Version:           "1.0",
		StaticNodes:       []*discovery.Node{},
		TokenFullName:     defaultTokenFullName,
		TokenShortName:    defaultTokenShortName,
		TokenAmount:       0,
		GenesisDifficulty: 8000000,
		GenesisAccounts:   map[common.Address]*big.Int{},
	}

	byteSubChainInfo, err := json.MarshalIndent(subChainInfo, "", "\t")
	if err != nil {
		return err
	}

	if err = common.SaveFile(filepath.Join(subChainJSONFileVale, "subChainTemplate.json"), byteSubChainInfo); err != nil {
		return err
	}

	fmt.Println("generate template json file for sub chain register successfully")
	return nil
}
