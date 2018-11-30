/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package listener

import (
	"io/ioutil"
	"strings"

	"fmt"
	"github.com/seeleteam/go-seele/accounts/abi"
	"github.com/seeleteam/go-seele/common/errors"
	seelelog "github.com/seeleteam/go-seele/log"
)

var (
	// ErrInvalidArguments is returned when NewContractEventABI arguments are invalid.
	ErrInvalidArguments = errors.New("the abiPath and eventName should not be empty")

	// ErrNoEvent is returned when the event is not in the specified abi.
	ErrNoEvent = errors.New("this contract has no such event")
)

var log = seelelog.GetLogger("eventListener")

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
			return nil, fmt.Errorf("This contract: %s, has no such event: %s", abiPath, eventName)
		}
		topic = event.Id().Hex()
		c.topicEventNames[topic] = eventName
	}

	return c, nil
}
