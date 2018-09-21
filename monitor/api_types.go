/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package monitor

// NodeInfo is the collection of meta information about a node that is displayed
// on the monitoring page.
type NodeInfo struct {
	Name       string `json:"name"`
	Node       string `json:"node"`
	Port       int    `json:"port"`
	NetVersion string `json:"netVersion"`
	Protocol   string `json:"protocol"`
	API        string `json:"api"`
	Os         string `json:"os"`
	OsVer      string `json:"os_v"`
	Client     string `json:"client"`
	History    bool   `json:"canUpdateHistory"`
	Shard      uint   `json:"shard"`
}

// NodeStats is the state information about the local node.
type NodeStats struct {
	Active  bool `json:"active"`
	Syncing bool `json:"syncing"`
	Mining  bool `json:"mining"`
	Peers   int  `json:"peers"`
}
