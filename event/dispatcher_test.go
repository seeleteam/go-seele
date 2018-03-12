/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package event

import (
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
)

var count int

func testfun0(e Event) { count++ }
func testfun1(e Event) { count++ }

func Test_EventManager(t *testing.T) {
	count = 0
	manager := NewEventManager()

	manager.AddListener(testfun0)
	manager.AddListener(testfun1)
	assert.Equal(t, len(manager.listeners), 2)

	manager.AddListener(testfun0) //test duplicate add
	event := EmptyEvent
	manager.Fire(event)
	assert.Equal(t, len(manager.listeners), 2)
	assert.Equal(t, count, 2)

	manager.RemoveListener(testfun1)
	manager.Fire(event)
	assert.Equal(t, count, 3)
}

func Test_ExecuteOnce(t *testing.T) {
	manager := NewEventManager()
	count = 0

	callback := func(e Event) {
		count++
	}

	callback2 := func(e Event) {
		count += 2
	}

	manager.AddOnceListener(callback)
	manager.AddOnceListener(callback2)
	assert.Equal(t, len(manager.listeners), 2)

	manager.Fire(EmptyEvent)
	assert.Equal(t, count, 3)
	manager.Fire(EmptyEvent)
	assert.Equal(t, count, 3)
	assert.Equal(t, len(manager.listeners), 0)
}

func Test_EventAsync(t *testing.T) {
	manager := NewEventManager()
	count = 0
	manager.AddAsyncListener(func(e Event) {
		time.Sleep(1)
		count += 1
	})

	manager.Fire(EmptyEvent)
	// async listener is sleeping
	assert.Equal(t, count, 0)
}

func Test_EventInstance(t *testing.T) {
	manager := NewEventManager()
	count = 0
	manager.AddListener(func(e Event) {
		v := e.(int)
		count += v
	})

	manager.Fire(2)
	assert.Equal(t, count, 2)

	manager.Fire(3)
	assert.Equal(t, count, 5)
}

func Test_EventOnceAndAsync(t *testing.T) {
	manager := NewEventManager()
	count = 0
	manager.AddAsyncOnceListener(func(e Event) {
		time.Sleep(1)
		count += 1
	})

	assert.Equal(t, len(manager.listeners), 1)

	manager.Fire(EmptyEvent)
	// async listener is sleeping
	assert.Equal(t, count, 0)

	time.Sleep(1)
	assert.Equal(t, count, 1)
	assert.Equal(t, len(manager.listeners), 0)
}