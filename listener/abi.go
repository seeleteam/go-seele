/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package listener

// The const strings below are system contracts.
// They will be taken place by product system contracts in future.
// Here are some example.
const subchainEvent1 = "getX"
const subchainABI1 = `
[
	{ "constant" : false, "inputs": [ { "name": "x", "type": "uint256" } ], "name": "set", "outputs": [], "payable": false, "stateMutability": "nonpayable", "type": "function" },
	{ "constant" : false, "inputs": [], "name": "get", "outputs": [ { "name": "", "type": "uint256" } ], "payable": false, "stateMutability": "nonpayable", "type": "function" },
	{ "inputs": [], "payable": false, "stateMutability": "nonpayable", "type": "constructor" },
	{ "anonymous": false, "inputs": [ { "indexed": false, "name": "", "type": "uint256" }, { "indexed": false, "name": "", "type": "uint256" } ], "name": "getX", "type": "event" },
	{ "anonymous": false, "inputs": [ { "indexed": false, "name": "", "type": "uint256" }, { "indexed": false, "name": "", "type": "uint256" } ], "name": "getY", "type": "event" }
]`

const subchainEvent2 = "getY"
const subchainABI2 = `
[
	{ "constant" : false, "inputs": [ { "name": "x", "type": "uint256" } ], "name": "set", "outputs": [], "payable": false, "stateMutability": "nonpayable", "type": "function" },
	{ "constant" : false, "inputs": [], "name": "get", "outputs": [ { "name": "", "type": "uint256" } ], "payable": false, "stateMutability": "nonpayable", "type": "function" },
	{ "inputs": [], "payable": false, "stateMutability": "nonpayable", "type": "constructor" },
	{ "anonymous": false, "inputs": [ { "indexed": false, "name": "", "type": "uint256" }, { "indexed": false, "name": "", "type": "uint256" } ], "name": "getX", "type": "event" },
	{ "anonymous": false, "inputs": [ { "indexed": false, "name": "", "type": "uint256" }, { "indexed": false, "name": "", "type": "uint256" } ], "name": "getY", "type": "event" }
]`

var refMp = map[string]string{
	subchainEvent1: subchainABI1,
	subchainEvent2: subchainABI2,
}

type config struct {
	events []eventCfg
}

type eventCfg struct {
	methodName string
	abi        string
}

func newConfig(ref map[string]string) *config {
	var (
		events []eventCfg
		cfg    config
	)
	for method, abi := range ref {
		if abi == "" {
			continue
		}
		var event eventCfg
		event.methodName = method
		event.abi = abi
		events = append(events, event)
	}

	cfg.events = events
	return &cfg
}
