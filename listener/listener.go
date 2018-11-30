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
	"github.com/seeleteam/go-seele/common/errors"
)

var (
	// ErrInvalidArguments is returned when NewContractEventABI arguments are invalid.
	ErrInvalidArguments = errors.New("the abiPath and eventName should not be empty")
)

// ContractEventABI represents contract event parser.
type ContractEventABI struct {
	// the map key is topic, the value is event name,
	// topic to compare with log's topic,
	// event name to label the event how to deal.
	topicEventNames map[string]string

	parser abi.ABI
}

// NewContractEventABI returns a ContractEventABI instance.
func NewContractEventABI(abiPath string, eventNames ...string) (*ContractEventABI, error) {
	if len(abiPath) == 0 || len(eventNames) == 0 {
		return nil, ErrInvalidArguments
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
		topicEventNames: make(map[string]string),
	}

	c.parser = parser

	var topic string
	for _, eventName := range eventNames {
		event, ok := parser.Events[eventName]
		if !ok {
			return nil, fmt.Errorf("event name %v not found in ABI file %v", eventName, string(file))
		}
		topic = event.Id().Hex()
		c.topicEventNames[topic] = eventName
	}

	return c, nil
}
