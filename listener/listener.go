/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package listener

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/seeleteam/go-seele/accounts/abi"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/common/errors"
	"github.com/seeleteam/go-seele/core/types"
)

var (
	// ErrInvalidArguments is returned when NewContractEventABI arguments are invalid.
	ErrInvalidArguments = errors.New("the abiPath, eventName and contract address cannot be empty")
)

// ContractEventABI represents contract event parser.
type ContractEventABI struct {
	// the map key is topic, the value is event name,
	// topic to compare with log's topic,
	// event name to label the event how to deal.
	topicEventNames map[common.Hash]string

	// contract address, to avoid different contracts have the same event name and arguments
	contract common.Address

	parser abi.ABI
}

// NewContractEventABI returns a ContractEventABI instance.
func NewContractEventABI(abiPath string, contract common.Address, eventNames ...string) (*ContractEventABI, error) {
	if len(abiPath) == 0 || len(eventNames) == 0 {
		return nil, ErrInvalidArguments
	}

	if contract.Equal(common.EmptyAddress) {
		return nil, ErrInvalidArguments
	}

	// ensure the contract address is EVM contract
	if !contract.IsEVMContract() {
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
		topicEventNames: make(map[common.Hash]string),
	}

	c.parser = parser

	for _, eventName := range eventNames {
		event, ok := parser.Events[eventName]
		if !ok {
			return nil, fmt.Errorf("event name %v not found in ABI file %v", eventName, abiPath)
		}
		c.topicEventNames[event.Id()] = eventName
	}

	return c, nil
}

// Event represents a contract event instance from Log.
type Event struct {
	Contract  common.Address
	EventName string
	Topic     common.Hash
	Arguments []interface{}
}

// GetEvents get events from receipts.
func (c *ContractEventABI) GetEvents(receipts []*types.Receipt) ([]*Event, error) {
	var events []*Event
	for _, receipt := range receipts {
		eventsSlice, err := c.GetEvent(receipt)
		if err != nil {
			return nil, err
		}

		events = append(events, eventsSlice...)
	}

	return events, nil
}

// GetEvent get events from receipt.
func (c *ContractEventABI) GetEvent(receipt *types.Receipt) ([]*Event, error) {
	var events []*Event
	if receipt.Failed {
		return nil, nil
	}

	for _, log := range receipt.Logs {
		if !log.Address.Equal(c.contract) || len(log.Topics) < 1 {
			continue
		}

		eventName, ok := c.topicEventNames[log.Topics[0]]
		if !ok {
			continue
		}

		event := &Event{
			Contract:  c.contract,
			EventName: eventName,
			Topic:     log.Topics[0],
		}

		var err error
		if log.Data != nil {
			// unnecessary to check whether parser.Events has the event name, we have check it before
			event.Arguments, err = c.parser.Events[eventName].Inputs.UnpackValues(log.Data)
			if err != nil {
				return nil, fmt.Errorf("event name %v not found in ABI file", eventName)
			}
		}

		events = append(events, event)
	}

	return events, nil
}
