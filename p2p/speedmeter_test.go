/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package p2p

import (
	"fmt"
	"testing"
	"time"

	"github.com/aristanetworks/goarista/monotime"
)

func Test_test1second(t *testing.T) {
	t1 := monotime.Now() / 1000
	fmt.Println("tick1=", t1)
	time.Sleep(1000)
	t2 := monotime.Now() / 1000
	fmt.Println("tick2=", t2, t2-t1)
	sp := NewSpeedMeter(100, 10)
	sp.Feed(128 * 1024)
	/*var i1 int64 = 10
	  var f1 = 1.0
	  r1 := i1 * f1
	  fmt.Println(i1 * f1)*/
}
