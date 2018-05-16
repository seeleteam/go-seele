/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package event

// EmptyEvent is an empty event
var EmptyEvent interface{}

// EventHandleMethod represents an event handler
type EventHandleMethod func(e Event)

// Event is the interface of events
type Event interface {
}

// eventListener is a struct which defines a function as a listener
type eventListener struct {
	// Callable is a callable function
	Callable        EventHandleMethod
	IsOnceListener  bool
	IsAsyncListener bool
}
