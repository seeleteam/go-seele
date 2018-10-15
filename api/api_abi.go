package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/seeleteam/go-seele/accounts/abi"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
)

// KeyABIHash is the hash key to storing abi to statedb
var KeyABIHash = common.StringToHash("KeyABIHash")

type seeleLog struct {
	Topics []string
	Event  string
	Args   []interface{}
}

func printReceiptByABI(api *TransactionPoolAPI, receipt *types.Receipt) (map[string]interface{}, error) {
	result, err := PrintableReceipt(receipt)
	if err != nil {
		return nil, err
	}

	// unpack result - todo: Since the methodName cannot be found now, it will be parsed in the next release.

	// unpack log
	if len(receipt.Logs) > 0 {
		logOuts := make([]string, 0)

		for _, log := range receipt.Logs {
			fmt.Println("log.Data：", log.Data)
			abiJSON, err := api.GetABI(log.Address)
			if err != nil {
				api.s.Log().Warn("the contract[%s] abi not found", log.Address.ToHex())
				return result, nil
			}

			// abiJSON := "[{\"constant\":true,\"inputs\":[],\"name\":\"creator\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"diceNumber\",\"type\":\"uint256\"},{\"name\":\"winValue\",\"type\":\"uint256\"}],\"name\":\"dice\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"destory\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"senders\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"diceNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"randNumber\",\"type\":\"uint256\"}],\"name\":\"lossAction\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"diceNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"randNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"winValue\",\"type\":\"uint256\"}],\"name\":\"winAction\",\"type\":\"event\"}]"
			parsed, err := abi.JSON(strings.NewReader(abiJSON))
			if err != nil {
				api.s.Log().Warn("invalid abiJSON '%s', err: %s", abiJSON, err)
				return result, nil
			}

			logOut, err := printLogByABI(log, parsed)
			if err != nil {
				api.s.Log().Warn("err: %s", err)
				return result, nil
			}

			logOuts = append(logOuts, logOut)
		}

		result["logs"] = logOuts
	}

	return result, nil
}

// GetABI according to contract address to get abi json string
func (api *TransactionPoolAPI) GetABI(contractAddr common.Address) (string, error) {
	statedb, err := api.s.ChainBackend().GetCurrentState()
	if err != nil {
		return "", err
	}

	abiBytes := statedb.GetData(contractAddr, KeyABIHash)
	if len(abiBytes) == 0 {
		return "", fmt.Errorf("the abi of contract '%s' not found", contractAddr.ToHex())
	}

	return string(abiBytes), nil
}

func printLogByABI(log *types.Log, parsed abi.ABI) (string, error) {
	seelelog := &seeleLog{}
	if len(log.Topics) < 1 {
		return "", nil
	}

	for _, topic := range log.Topics {
		seelelog.Topics = append(seelelog.Topics, topic.Hex())
	}

	for _, event := range parsed.Events {
		if event.Id().Hex() == seelelog.Topics[0] {
			seelelog.Event = event.Name
			break
		}
	}

	var err error
	seelelog.Args, err = parsed.Events[seelelog.Event].Inputs.UnpackValues(log.Data)
	if err != nil {
		return "", err
	}

	encoded, err := json.Marshal(seelelog)
	if err != nil {
		return "", err
	}

	return string(encoded), nil
}
