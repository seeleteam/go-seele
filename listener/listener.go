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
	"github.com/seeleteam/go-seele/common/errors"
	seelelog "github.com/seeleteam/go-seele/log"
)

var (
	// ErrNoEventToListen is returned when NewListener arguments are nil
	ErrNoEventToListen = errors.New("no event to listen")

	// ErrInvalidArguments is returned when NewListener arguments are invalid
	ErrInvalidArguments = errors.New("the format should be 'abi path' + ',' + 'event name1' + ',' + 'event name2' + ... + 'event namen'")
)

var log = seelelog.GetLogger("eventListener")

// ABIParser represents abi parser.
type ABIParser struct {
	EventName string
	Parser    abi.ABI
}

// Listener represents contract event listener.
type Listener struct {
	ABIParsers map[string]ABIParser
}

// ABIInfo represents the information of the abi and event.
type abiInfo struct {
	abi       string
	eventName string
}
type config struct {
	events []abiInfo
}

// NewListener returns an initialized Listener.
// The input arguments format should be 'abi path' + ',' + 'event name1' + ',' + 'event name2' + ... + 'event namen'.
func NewListener(abiEvents []string) (*Listener, error) {
	if len(abiEvents) == 0 {
		return nil, ErrNoEventToListen
	}

	var (
		args  []string
		event abiInfo
		cfg   config
	)
	for _, abi := range abiEvents {
		args = strings.Split(abi, ",")
		if len(args) < 2 {
			return nil, ErrInvalidArguments
		}

		file, err := ioutil.ReadFile(args[0])
		if err != nil {
			return nil, fmt.Errorf("failed to read abi file, err: %s", err.Error())
		}

		event.abi = string(file)
		for _, eventName := range args[1:] {
			event.eventName = eventName
			cfg.events = append(cfg.events, event)
		}
	}

	l := &Listener{
		ABIParsers: make(map[string]ABIParser),
	}

	abiParserMp := make(map[string]ABIParser)
	for _, event := range cfg.events {
		parser, err := abi.JSON(strings.NewReader(event.abi))
		if err != nil {
			return nil, errors.NewStackedError(err, "failed to parse abi")
		}

		topic := parser.Events[event.eventName].Id().Hex()

		ABIParser := ABIParser{
			EventName: event.eventName,
			Parser:    parser,
		}
		abiParserMp[topic] = ABIParser
	}

	l.ABIParsers = abiParserMp

	return l, nil
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
	EventName string
	Topic     string
	Arguments []interface{}
}

// GetGroupLogs converts the logs of receipts to map[string][]*Log.
func GetGroupLogs(receipts []*Receipt) map[string][]*Log {
	logGroup := make(map[string][]*Log)
	for _, receipt := range receipts {
		if receipt.Failed || len(receipt.Logs) == 0 {
			continue
		}

		for _, log := range receipt.Logs {
			logGroup[log.Topic] = append(logGroup[log.Topic], log)
		}
	}

	return logGroup
}

// GetLogsByTopic returns []*Log by topic.
func GetLogsByTopic(topic string, receipts []*Receipt) []*Log {
	var logs []*Log
	for _, receipt := range receipts {
		if receipt.Failed || len(receipt.Logs) == 0 {
			continue
		}

		for _, log := range receipt.Logs {
			if topic == log.Topic {
				logs = append(logs, log)
			}
		}
	}

	return logs
}

// ParseLogGroupToEvents parse the Logs in map[string][]*Log to Events.
func (l *Listener) ParseLogGroupToEvents(lg map[string][]*Log) ([]Event, error) {
	var events []Event

	for topic, parser := range l.ABIParsers {
		logs, ok := lg[topic]
		if !ok || logs == nil {
			continue
		}

		eventslice, err := parseLogsToEvents(topic, parser, logs)
		if err != nil {
			return nil, err
		}

		events = append(events, eventslice...)
	}

	return events, nil
}

// ParseLogsToEventsByTopic parse the Logs in []*Log to Events by specified topic.
func (l *Listener) ParseLogsToEventsByTopic(topic string, logs []*Log) ([]Event, error) {
	parser, ok := l.ABIParsers[topic]
	if !ok {
		return nil, fmt.Errorf("the topic has no parser: %s", topic)
	}

	return parseLogsToEvents(topic, parser, logs)
}

func parseLogsToEvents(topic string, parser ABIParser, logs []*Log) ([]Event, error) {
	var events []Event

	for _, l := range logs {
		var event Event
		if len(l.Data) != 0 {
			args := make([]string, len(l.Data))
			for i, data := range l.Data {
				args[i] = data[2:]
			}

			b, err := hex.DecodeString(strings.Join(args, ""))
			if err != nil {
				return nil, errors.NewStackedError(err, "abi decode hex string to byte failed")
			}

			event.Arguments, err = parser.Parser.Events[parser.EventName].Inputs.UnpackValues(b)
			if err != nil {
				log.Error("abi decode input args failed, %s", err.Error())
				return nil, err
			}
		}

		event.EventName = parser.EventName
		event.Topic = topic
		events = append(events, event)
	}

	return events, nil
}
