package eventdispatcher

var EmptyEvent interface{}

type EventHandleMethod func(e Event)

// Event interface
type Event interface {
}

// Listener type for defining functions as listeners
type Listener struct {
	//Callable call function
	Callable func(e Event)
}
