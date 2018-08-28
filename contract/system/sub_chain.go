/**
* @file
* @copyright defined in go-seele/LICENSE
 */

package system

const (
	cmdSubChainRegister byte = iota // register a sub-chain.
	cmdSubChainQuery

	gasSubChainRegister = uint64(100000) // gas to register a sub-chain.
	gasSubChainQuery    = uint64(200000) // gas to query sub-chain information.
)

var (
	subChainCommands = map[byte]*cmdInfo{
		cmdSubChainRegister: &cmdInfo{gasSubChainRegister, registerSubChain},
		cmdSubChainQuery:    &cmdInfo{gasSubChainQuery, querySubChain},
	}
)

func registerSubChain(jsonRegInfo []byte, context *Context) ([]byte, error) {
	return nil, nil
}

func querySubChain(subChainName []byte, context *Context) ([]byte, error) {
	return nil, nil
}
