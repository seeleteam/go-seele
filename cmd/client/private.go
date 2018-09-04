/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"github.com/seeleteam/go-seele/cmd/util"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/p2p"
	"github.com/seeleteam/go-seele/rpc2"
)

func GetTxPoolContentAction(client *rpc.Client) (interface{}, error) {
	var result map[string][]map[string]interface{}
	err := client.Call(&result, "debug_getTxPoolContent")
	return result, err
}

func GetTxPoolTxCountAction(client *rpc.Client) (interface{}, error) {
	var result uint64
	err := client.Call(&result, "debug_getTxPoolTxCount")
	return result, err
}

func StartMinerAction(client *rpc.Client) (interface{}, error) {
	var result bool
	err := client.Call(&result, "miner_start", threadsValue)
	return result, err
}

func StopMinerAction(client *rpc.Client) (interface{}, error) {
	var result bool
	err := client.Call(&result, "miner_stop")
	return result, err
}

func SetMinerThreadsAction(client *rpc.Client) (interface{}, error) {
	var result bool
	err := client.Call(&result, "miner_setThreads", threadsValue)
	return result, err
}

func GetMinerThreadsAction(client *rpc.Client) (interface{}, error) {
	var result int
	err := client.Call(&result, "miner_getThreads")
	return result, err
}

func GetMinerStatusAction(client *rpc.Client) (interface{}, error) {
	var result string
	err := client.Call(&result, "miner_status")
	return result, err
}

func GetMinerHashrateAction(client *rpc.Client) (interface{}, error) {
	var result uint64
	err := client.Call(&result, "miner_hashrate")
	return result, err
}

func SetMinerCoinbaseAction(client *rpc.Client) (interface{}, error) {
	var result bool
	err := client.Call(&result, "miner_setCoinbase", coinbaseValue)
	return result, err
}

func GetMinerCoinbaseAction(client *rpc.Client) (interface{}, error) {
	var result common.Address
	err := client.Call(&result, "miner_getCoinbase")
	return result, err
}

func GetBlockTransactionCountAction(client *rpc.Client) (interface{}, error) {
	var result int
	var err error

	if hashValue != "" {
		err = client.Call(&result, "txpool_getBlockTransactionCountByHash", hashValue)
	} else {
		err = client.Call(&result, "txpool_getBlockTransactionCountByHeight", heightValue)
	}

	return result, err
}

func GetTransactionAction(client *rpc.Client) (interface{}, error) {
	var result map[string]interface{}
	var err error

	if hashValue != "" {
		err = client.Call(&result, "txpool_getTransactionByBlockHashAndIndex", hashValue, indexValue)
	} else {
		err = client.Call(&result, "txpool_getTransactionByBlockHeightAndIndex", heightValue, indexValue)
	}

	return result, err
}

func GetReceiptByTxHashAction(client *rpc.Client) (interface{}, error) {
	var result map[string]interface{}
	err := client.Call(&result, "txpool_getReceiptByTxHash", hashValue)
	return result, err
}

func GetTransactionByHashAction(client *rpc.Client) (interface{}, error) {
	return util.GetTransactionByHash(client, hashValue)
}

func GetPendingDebtsAction(client *rpc.Client) (interface{}, error) {
	var result []*types.Debt
	err := client.Call(&result, "txpool_getPendingDebts")
	return result, err
}

func GetDebtByHashAction(client *rpc.Client) (interface{}, error) {
	var result map[string]interface{}
	err := client.Call(&result, "txpool_getDebtByHash", hashValue)
	return result, err
}

func GetPendingTransactionsAction(client *rpc.Client) (interface{}, error) {
	var result []map[string]interface{}
	err := client.Call(&result, "txpool_getPendingTransactions")
	return result, err
}

func GetPeerCountAction(client *rpc.Client) (interface{}, error) {
	var result int
	err := client.Call(&result, "network_getPeerCount")
	return result, err
}

// GetPeersInfo get peers information
func GetPeersInfo(client *rpc.Client) (interface{}, error) {
	var result []p2p.PeerInfo
	err := client.Call(&result, "network_getPeersInfo")
	return result, err
}

// GetNetworkVersion get current network version
func GetNetworkVersion(client *rpc.Client) (interface{}, error) {
	var result uint64
	err := client.Call(&result, "network_getNetworkVersion")
	return result, err
}

// GetProtocolVersion get seele protocol version
func GetProtocolVersion(client *rpc.Client) (interface{}, error) {
	var result uint
	err := client.Call(&result, "network_getProtocolVersion")
	return result, err
}

// GetDumpHeap dump heap for profiling
func GetDumpHeap(client *rpc.Client) (interface{}, error) {
	var result string
	err := client.Call(&result, "debug_dumpHeap", dumpFileValue, gcBeforeDump)
	return result, err
}
