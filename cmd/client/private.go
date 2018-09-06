/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package main

import (
	"github.com/seeleteam/go-seele/rpc2"
)

func getBlockTransactionCountAction(client *rpc.Client) (interface{}, error) {
	var result int
	var err error

	if hashValue != "" {
		err = client.Call(&result, "txpool_getBlockTransactionCountByHash", hashValue)
	} else {
		err = client.Call(&result, "txpool_getBlockTransactionCountByHeight", heightValue)
	}

	return result, err
}

func getTransactionAction(client *rpc.Client) (interface{}, error) {
	var result map[string]interface{}
	var err error

	if hashValue != "" {
		err = client.Call(&result, "txpool_getTransactionByBlockHashAndIndex", hashValue, indexValue)
	} else {
		err = client.Call(&result, "txpool_getTransactionByBlockHeightAndIndex", heightValue, indexValue)
	}

	return result, err
}
