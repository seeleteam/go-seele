/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package event

import (
	"reflect"
	"sync"
)

// EventManager interface defines the event manager behavior
// Note, it is thread safe
type EventManager struct {
	lock      sync.RWMutex
	listeners []Listener
}

// Fire fire the event and returns it after all listeners do
// their jobs.
func (h *EventManager) Fire(e Event) {
	h.lock.RLock()
	defer h.lock.RUnlock()
	for _, l := range h.listeners {
		l.Callable(e)
	}

	h.removeOnceListener()
}

// AddListener registers a listener.
// If there already has a same listener (same method pointer), we will not add it
func (h *EventManager) AddListener(listener Listener) {
	h.lock.Lock()
	defer h.lock.Unlock()
	if index := h.find(listener); index != -1 {
		return
	}

	h.listeners = append(h.listeners, listener)
}

// RemoveListener removes the registered event listener for given event name.
func (h *EventManager) RemoveListener(listener Listener) {
	h.lock.Lock()
	defer h.lock.Unlock()
	index := h.find(listener)
	if index == -1 {
		return
	}

	h.listeners = append(h.listeners[:index], h.listeners[index+1:]...)
}

func (h *EventManager) removeOnceListener() {
	listener := make([]Listener, 0, len(h.listeners))
	for _, l := range h.listeners{
		if !l.IsOnceListener {
			listener = append(listener, l)
		}
	}

	h.listeners = listener
}

// find find listener already in the manager
// return -1 not found, otherwise return the index of the listener
func (h *EventManager) find(listener Listener) int {
	p := reflect.ValueOf(listener.Callable).Pointer()

	for i, l := range h.listeners {
		lp := reflect.ValueOf(l.Callable).Pointer()
		if lp == p {
			return i
		}
	}

	return -1
}

// NewEventManager creates a new instance of event manager
func NewEventManager() *EventManager {
	return &EventManager{
		listeners: make([]Listener, 0),
	}
}
