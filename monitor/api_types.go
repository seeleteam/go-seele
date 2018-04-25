/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package monitor

import (
	"math/big"

	"github.com/seeleteam/go-seele/common"
)

// NodeInfo is the collection of metainformation about a node that is displayed
// on the monitoring page.
type NodeInfo struct {
	Name       string `json:"name"`
	Node       string `json:"node"`
	Port       int    `json:"port"`
	NetVersion uint64 `json:"netVersion"`
	Protocol   string `json:"protocol"`
	API        string `json:"api"`
	Os         string `json:"os"`
	OsVer      string `json:"os_v"`
	Client     string `json:"client"`
	History    bool   `json:"canUpdateHistory"`
}

// NodeStats is the information about the local node.
type NodeStats struct {
	Active  bool `json:"active"`
	Syncing bool `json:"syncing"`
	Mining  bool `json:"mining"`
	Peers   int  `json:"peers"`
}

// CurrentBlock is the informations about the best block
type CurrentBlock struct {
	HeadHash  common.Hash    `json:"headHash"`
	Height    uint64         `json:"height"`
	Timestamp *big.Int       `json:"timestamp"`
	Difficult *big.Int       `json:"difficult"`
	Creator   common.Address `json:"creator"`
	TxCount   int            `json:"txcount"`
}
