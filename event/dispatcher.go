/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package event

import (
	"reflect"
	"sync"
)

// EventHandler interface defines the event handler behavior
type EventHandler struct {
	lock      sync.RWMutex
	listeners []Listener
}

// Fire fire the event and returns it after all listeners do
// their jobs.
func (h *EventHandler) Fire(e Event) {
	h.lock.RLock()
	defer h.lock.RUnlock()
	for _, l := range h.listeners {
		go l.Callable(e)
	}
}

// AddListener registers a listener.
// If there already has a same listener (same method pointer), we will not add it
func (h *EventHandler) AddListener(listener Listener) {
	if index := h.find(listener); index != -1 {
		return
	}

	h.lock.Lock()
	defer h.lock.Unlock()
	h.listeners = append(h.listeners, listener)
}

// RemoveListener removes the registered event listener for given event name.
func (h *EventHandler) RemoveListener(listener Listener) {
	index := h.find(listener)
	if index == -1 {
		return
	}

	h.lock.Lock()
	defer h.lock.Unlock()
	h.listeners = append(h.listeners[:index], h.listeners[index+1:]...)
}

// find find listener already in the handler
// return -1 not found, otherwise return the index of the listener
func (h *EventHandler) find(listener Listener) int {
	p := reflect.ValueOf(listener.Callable).Pointer()

	h.lock.RLock()
	defer h.lock.RUnlock()
	for i, l := range h.listeners {
		lp := reflect.ValueOf(l.Callable).Pointer()
		if lp == p {
			return i
		}
	}

	return -1
}

// NewEventHandler creates a new instance of event handler
func NewEventHandler() *EventHandler {
	return &EventHandler{
		listeners: make([]Listener, 0),
	}
}
