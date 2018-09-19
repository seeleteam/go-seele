/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/hexutil"
	"github.com/seeleteam/go-seele/common/keystore"
	"github.com/seeleteam/go-seele/contract/system"
	"github.com/seeleteam/go-seele/core"
	"github.com/seeleteam/go-seele/crypto"
	"github.com/seeleteam/go-seele/log/comm"
	"github.com/seeleteam/go-seele/metrics"
	"github.com/seeleteam/go-seele/node"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/rpc"
	"github.com/urfave/cli"
)

var (
	errInvalidVersion        = errors.New("invalid subchain version")
	errInvalidTokenFullName  = errors.New("invalid subchain token full name")
	errInvalidTokenShortName = errors.New("invalid subchain token short name")
	errInvalidTokenAmount    = errors.New("invalid subchain token amount")
	errSubChainInfo          = errors.New("failed to get sub-chain information")
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

	if len(subChain.TokenFullName) == 0 {
		return nil, nil, errInvalidTokenFullName
	}

	if len(subChain.TokenShortName) == 0 {
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

	config, err := getConfigFromSubChain(subChainInfo)
	if err != nil {
		return err
	}

	// save accounts file
	byteAccounts, err := json.MarshalIndent(subChainInfo.GenesisAccounts, "", "\t")
	if err != nil {
		return err
	}
	err = common.SaveFile(filepath.Join(outPutValue, "accounts.json"), byteAccounts)
	if err != nil {
		return err
	}

	// save node file
	byteNode, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return err
	}
	err = common.SaveFile(filepath.Join(outPutValue, "node.json"), byteNode)
	if err != nil {
		return err
	}

	fmt.Println("generate subchain config files successfully")
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

func getKeyFromFile() (*keystore.Key, error) {
	pass, err := common.GetPassword()
	if err != nil {
		return nil, fmt.Errorf("failed to get password %s", err)
	}

	key, err := keystore.GetKey(keyFileValue, pass)
	if err != nil {
		return nil, fmt.Errorf("invalid key file. it should be a private key: %s", err)
	}

	return key, nil
}

func getSubChainFromReceipt(client *rpc.Client) (*system.SubChainInfo, error) {
	if err := system.ValidateDomainName([]byte(nameValue)); err != nil {
		return nil, err
	}
	payloadBytes := append([]byte{system.CmdSubChainQuery}, []byte(nameValue)...)
	mapReceipt, err := util.CallContract(client, system.SubChainContractAddress.ToHex(), hexutil.BytesToHex(payloadBytes), -1)
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

	var subChainInfo system.SubChainInfo
	err = json.Unmarshal(bytesSubChainInfo, &subChainInfo)
	if err != nil {
		return nil, err
	}

	return &subChainInfo, nil
}

func getConfigFromSubChain(subChainInfo *system.SubChainInfo) (*util.Config, error) {
	key, err := getKeyFromFile()
	if err != nil {
		return nil, err
	}

	config := &util.Config{}
	config.BasicConfig = node.BasicConfig{
		Name:     subChainInfo.Name,
		Version:  subChainInfo.Version,
		DataDir:  subChainInfo.Name,
		RPCAddr:  "0.0.0.0:8027",
		Coinbase: key.Address.ToHex(),
		SyncMode: "full",
	}

	config.P2PConfig = p2p.Config{
		NetworkID:     1,
		ListenAddr:    "0.0.0.0:8057",
		StaticNodes:   subChainInfo.StaticNodes,
		SubPrivateKey: hexutil.BytesToHex(crypto.FromECDSA(key.PrivateKey)),
	}

	config.LogConfig = comm.LogConfig{
		PrintLog: true,
	}

	config.HTTPServer = node.HTTPServer{
		HTTPAddr:      "127.0.0.1:8036",
		HTTPCors:      []string{"*"},
		HTTPWhiteHost: []string{"*"},
	}

	config.WSServerConfig = node.WSServerConfig{
		Address:      "127.0.0.1:8046",
		CrossOrigins: []string{"*"},
	}

	config.MetricsConfig = &metrics.Config{
		Addr:     "127.0.0.1:8087",
		Duration: time.Duration(10),
		Database: "influxdb",
		Username: "test",
		Password: "test123",
	}

	config.GenesisConfig = core.GenesisInfo{
		Difficult:   int64(subChainInfo.GenesisDifficulty),
		ShardNumber: key.Address.Shard(),
	}

	return config, nil
}
