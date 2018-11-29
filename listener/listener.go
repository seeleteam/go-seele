/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package listener

import (
	"fmt"
	"io/ioutil"
	"strings"
	"sync/atomic"

	"github.com/seeleteam/go-seele/accounts/abi"
	"github.com/seeleteam/go-seele/common/errors"
	seelelog "github.com/seeleteam/go-seele/log"
)

var (
	// ErrInvalidArguments is returned when NewListener arguments are invalid.
	ErrInvalidArguments = errors.New("the abiPath and eventName should be not empty")

	// ErrListenerIsRunning is returned when start the Listener but the Listener is running.
	ErrListenerIsRunning = errors.New("listener is running")

	// ErrNoEvent is returned when the event is not in the specified abi.
	ErrNoEvent = errors.New("This contract has no such event")
)

var log = seelelog.GetLogger("eventListener")

// Listener represents contract event listener.
type Listener struct {
	running int32

	abiPath    string
	eventNames []string
	topics     map[string]struct{}
	parser     abi.ABI
}

func (l *Listener) String() string {
	var topics []string
	for topic := range l.topics {
		topics = append(topics, topic)
	}

	return strings.Join(topics, ",")
}

// NewListener returns a Listener instance.
func NewListener(abiPath string, eventNames ...string) (*Listener, error) {
	if abiPath == "" || len(eventNames) == 0 {
		return nil, ErrInvalidArguments
	}

	log.Info("New Listener with abi path: %s, events: %v", abiPath, eventNames)

	return &Listener{
		abiPath:    abiPath,
		eventNames: eventNames,
		topics:     make(map[string]struct{}),
	}, nil
}

// Start parses the abi info to topics and parser, and begin the listener.
func (l *Listener) Start() error {
	if atomic.LoadInt32(&l.running) == 1 {
		log.Info("Listener is running")
		return ErrListenerIsRunning
	}

	if !atomic.CompareAndSwapInt32(&l.running, 0, 1) {
		log.Info("Another goroutine has already started the listener")
		return nil
	}

	file, err := ioutil.ReadFile(l.abiPath)
	if err != nil {
		atomic.StoreInt32(&l.running, 0)
		return fmt.Errorf("failed to read abi file, err: %s", err.Error())
	}

	parser, err := abi.JSON(strings.NewReader(string(file)))
	if err != nil {
		atomic.StoreInt32(&l.running, 0)
		return errors.NewStackedError(err, "failed to parse abi")
	}

	l.parser = parser

	var topic string
	for _, eventName := range l.eventNames {
		event, ok := parser.Events[eventName]
		if !ok {
			atomic.StoreInt32(&l.running, 0)
			log.Error("This contract: %s, has no such event: %s", l.abiPath, eventName)
			return ErrNoEvent
		}
		topic = event.Id().Hex()
		l.topics[topic] = struct{}{}
	}

	log.Info("Listener running with abi path: %s, topics: %v", l.abiPath, l)

	return nil
}

// Stop is used to stop the Listener
func (l *Listener) Stop() {
	atomic.StoreInt32(&l.running, 0)
	l.topics = make(map[string]struct{})
}
