/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package event

import (
	"github.com/magiconair/properties/assert"
	"testing"
)

var count int

func testfun0(e Event) { count++ }
func testfun1(e Event) { count++ }

func Test_EventManager(t *testing.T) {
	count = 0
	listener1 := Listener{Callable: testfun0}
	listener2 := Listener{Callable: testfun1}

	manager := NewEventManager()

	manager.AddListener(listener1)
	manager.AddListener(listener2)
	assert.Equal(t, len(manager.listeners), 2)

	manager.AddListener(listener1) //test duplicate add
	event := EmptyEvent
	manager.Fire(event)

	assert.Equal(t, len(manager.listeners), 2)
	assert.Equal(t, count, 2)

	manager.RemoveListener(listener2)

	manager.Fire(event)
	assert.Equal(t, count, 3)
}

func Test_ExecuteOnce(t *testing.T) {
	handler := NewEventManager()
	count = 0

	listener := Listener{
		Callable: func(e Event) {
			count++
		},
		IsOnceListener: true,
	}

	listener2 := Listener {
		Callable: func(e Event) {
			count += 2
		},
		IsOnceListener: true,
	}

	handler.AddListener(listener)
	handler.AddListener(listener2)
	assert.Equal(t, len(handler.listeners), 2)

	handler.Fire(EmptyEvent)
	assert.Equal(t, count, 3)
	handler.Fire(EmptyEvent)
	assert.Equal(t, count, 3)
	assert.Equal(t, len(handler.listeners), 0)
}
