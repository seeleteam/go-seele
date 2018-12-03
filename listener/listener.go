/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package listener

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/seeleteam/go-seele/accounts/abi"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
)

var (
	// ErrInvalidArguments is returned when NewContractEventABI arguments are invalid.
	ErrInvalidArguments = errors.New("the abiPath or eventName or contract address should not be empty")
)

// ContractEventABI represents contract event parser.
type ContractEventABI struct {
	// the map key is topic, the value is event name,
	// topic to compare with log's topic,
	// event name to label the event how to deal.
	topicEventNames map[string]string

	// contract address, to avoid different contracts have the same event name and arguments
	contract string

	parser abi.ABI
}

// NewContractEventABI returns a ContractEventABI instance.
func NewContractEventABI(abiPath string, contract string, eventNames ...string) (*ContractEventABI, error) {
	if len(abiPath) == 0 || len(contract) == 0 || len(eventNames) == 0 {
		return nil, ErrInvalidArguments
	}

	// ensure the contract address is EVM contract
	tempAddress, err := common.HexToAddress(contract)
	if err != nil {
		return nil, errors.NewStackedError(err, "invalid contract address,")
	}

	if !tempAddress.IsEVMContract() {
		return nil, fmt.Errorf("the address is not EVM contract, %v", contract)
	}

	file, err := ioutil.ReadFile(abiPath)
	if err != nil {
		return nil, errors.NewStackedError(err, "failed to read abi file")
	}

	parser, err := abi.JSON(strings.NewReader(string(file)))
	if err != nil {
		return nil, errors.NewStackedError(err, "failed to parse abi")
	}

	c := &ContractEventABI{
		contract:        contract,
		topicEventNames: make(map[string]string),
	}

	c.parser = parser

	var topic string
	for _, eventName := range eventNames {
		event, ok := parser.Events[eventName]
		if !ok {
			return nil, fmt.Errorf("event name %v not found in ABI file %v", eventName, abiPath)
		}
		topic = event.Id().Hex()
		c.topicEventNames[topic] = eventName
	}

	return c, nil
}

// Receipt represents a receipt instance from main chain block.
type Receipt struct {
	Failed bool
	Logs   []*Log
}

// Log represents a log instance from main chain block.
type Log struct {
	Address string
	Data    []string
	Topic   string
}

// Event represents a contract event instance from Log.
type Event struct {
	Contract  string
	EventName string
	Topic     string
	Arguments []interface{}
}

// GetEventsFromReceipts get events from receipts.
func (c *ContractEventABI) GetEventsFromReceipts(receipts []*Receipt) ([]*Event, error) {
	var events []*Event
	for _, receipt := range receipts {
		if receipt.Failed {
			continue
		}

		for _, log := range receipt.Logs {
			if log.Address != c.contract {
				continue
			}

			eventName, ok := c.topicEventNames[log.Topic]
			if !ok {
				continue
			}

			var event Event
			if len(log.Data) != 0 {
				args := make([]string, len(log.Data))
				for i, data := range log.Data {
					args[i] = data[2:]
				}

				b, err := hex.DecodeString(strings.Join(args, ""))
				if err != nil {
					return nil, errors.NewStackedError(err, "failed to decode hex string to bytes")
				}

				// unnecessary to check whether parser.Events has the event name, we have check it before
				event.Arguments, err = c.parser.Events[eventName].Inputs.UnpackValues(b)
				if err != nil {
					return nil, fmt.Errorf("event name %v not found in ABI file", eventName)
				}
			}

			event.Contract = c.contract
			event.EventName = eventName
			event.Topic = log.Topic
			events = append(events, &event)
		}
	}

	return events, nil
}
