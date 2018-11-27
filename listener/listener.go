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
	// ErrEventABILoadFailed is returned when abi string can't be loaded
	ErrEventABILoadFailed = errors.New("abi load failed")
)

var log = seelelog.GetLogger("event-listener")

// Listener represents contract event listener.
type Listener struct {
	ABIParsers map[ABIInfo]abi.ABI
}

// ABIInfo represents the information of the abi and event.
type ABIInfo struct {
	ABI       string
	EventName string
	Topic     string
}

func (abi ABIInfo) String() string {
	return fmt.Sprintf("abi:%s, Event:%s, Topic:%s", abi.ABI, abi.EventName, abi.Topic)
}

type config struct {
	events []abiEvent
}

type abiEvent struct {
	abi       string
	eventName string
}

// NewListener returns an initialized Listener.
// The input arguments format should be 'abi path' + ',' + 'event name1' + ',' + 'event name2' + ... + 'event namen'.
func NewListener(abiEvents []string) (*Listener, error) {
	if len(abiEvents) == 0 {
		return nil, fmt.Errorf("no event to listen")
	}

	var (
		s     []string
		event abiEvent
		cfg   config
	)
	for _, abi := range abiEvents {
		s = strings.Split(abi, ",")
		if len(s) < 2 {
			return nil, fmt.Errorf("the format should be 'abi path' + ',' + 'event name1' + ',' + 'event name2' + ... + 'event namen'")
		}

		file, err := ioutil.ReadFile(s[0])
		if err != nil {
			return nil, fmt.Errorf("failed to read abi file, err: %s", err.Error())
		}

		event.abi = string(file)
		for _, eventName := range s[1:] {
			event.eventName = eventName
			cfg.events = append(cfg.events, event)
		}
	}

	l := &Listener{
		ABIParsers: make(map[ABIInfo]abi.ABI),
	}

	if err := l.getABIParsers(&cfg); err != nil {
		return nil, err
	}

	return l, nil
}

func (l *Listener) getABIParsers(cfg *config) error {
	abiParserMp := make(map[ABIInfo]abi.ABI)
	for _, event := range cfg.events {
		parser, err := abi.JSON(strings.NewReader(event.abi))
		if err != nil {
			log.Error("read abi failed, %s, %s", event.abi, err.Error())
			return ErrEventABILoadFailed
		}

		topic := parser.Events[event.eventName].Id().Hex()
		key := ABIInfo{
			ABI:       event.abi,
			EventName: event.eventName,
			Topic:     topic,
		}

		abiParserMp[key] = parser
	}

	l.ABIParsers = abiParserMp

	return nil
}

// GetABIInfoByABIAndEventName returns ABIInfo struct by abi and event name.
func (l *Listener) GetABIInfoByABIAndEventName(abi *string, eventName string) (ABIInfo, error) {
	for key := range l.ABIParsers {
		if key.ABI == *abi && key.EventName == eventName {
			return key, nil
		}
	}

	return ABIInfo{}, fmt.Errorf("there is no ABIInfo for abi: %s, eventName: %s", abi, eventName)
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
	At        ABIInfo
	Arguments []interface{}
}

// GetGroupLogs converts the logs of receipts to map[string][]*Log.
func GetGroupLogs(receipts []*Receipt) map[string][]*Log {
	if len(receipts) == 0 {
		return nil
	}

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
	if len(receipts) == 0 {
		return nil
	}

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

	for at, parser := range l.ABIParsers {
		logs, ok := lg[at.Topic]
		if !ok || logs == nil {
			continue
		}

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

				event.Arguments, err = parser.Events[at.EventName].Inputs.UnpackValues(b)
				if err != nil {
					log.Error("abi decode input args failed, %s", err.Error())
					return nil, err
				}
			}

			event.At = at
			events = append(events, event)
		}
	}

	return events, nil
}

// ParseLogsToEventsByABIInfo parse the Logs in map[string][]*Log to Events by specified ABIInfo.
func (l *Listener) ParseLogsToEventsByABIInfo(info ABIInfo, logs []*Log) ([]Event, error) {
	var events []Event

	parser, ok := l.ABIParsers[info]
	if !ok {
		return nil, fmt.Errorf("the abi has no parser: %v", info)
	}

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

			event.Arguments, err = parser.Events[info.EventName].Inputs.UnpackValues(b)
			if err != nil {
				log.Error("abi decode input args failed, %s", err.Error())
				return nil, err
			}
		}

		event.At = info
		events = append(events, event)
	}

	return events, nil
}
