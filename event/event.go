/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package event

// EmptyEvent empty event
var EmptyEvent interface{}

// EventHandleMethod event handler method
type EventHandleMethod func(e Event)

// Event interface
type Event interface {
}

// eventListener type for defining functions as listeners
type eventListener struct {
	// Callable call function
	Callable        EventHandleMethod
	IsOnceListener  bool
	IsAsyncListener bool
}
