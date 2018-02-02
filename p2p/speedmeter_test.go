/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package p2p

import (
    "fmt"
    "testing"
    "time"
)

func Test_speedMeter(t *testing.T) {
    sp := NewSpeedMeter(100, 10)
    sp.Feed(128 * 1024)
    time.Sleep(300)
    sp.Feed(128 * 1024)
    fmt.Println("speedMeter:", sp.GetRate())
}
