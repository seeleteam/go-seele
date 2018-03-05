/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package event

import (
	"fmt"
	"testing"
)

func testfun0(e Event) { fmt.Println("hello-0") }
func testfun1(e Event) { fmt.Println("hello-1") }

func Test_Handler(t *testing.T) {
	listener1 := Listener{Callable: testfun0}
	listener2 := Listener{Callable: testfun1}

	handler := NewEventHandler()

	handler.AddListener(listener1)
	handler.AddListener(listener2)
	handler.AddListener(listener1) //test duplicate add
	event := EmptyEvent
	fmt.Println("test 1")
	handler.Fire(event)

	handler.RemoveListener(listener2)

	fmt.Println("test 3")
	handler.Fire(event)
}

func Test_ExecuteOnce(t *testing.T) {
	handler := NewEventHandler()

	var listener Listener
	listener = Listener{
		Callable: func(e Event) {
			fmt.Println("execution once")
			handler.RemoveListener(listener)
		},
	}

	fmt.Println("test 1")
	handler.Fire(EmptyEvent)
	fmt.Println("test 2")
	handler.Fire(EmptyEvent)
}
