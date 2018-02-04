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

func Test_NewSpeedMeter(t *testing.T) {
    var step uint64 = 5
    var itemsNum uint = 10
    v1 := NewSpeedMeter(step, itemsNum)

    if v1.step != step || v1.itemsNum != itemsNum {
        fmt.Println("NewSpeedMeter Init", v1.step, v1.itemsNum)
        t.Fail()
    }
}

func Test_SpeedMeterFeedAndGetRate(t *testing.T) {
    var step uint64 = 5
    var itemsNum uint = 10
    v1 := NewSpeedMeter(step, itemsNum)

    for _, ele := range v1.itemArr {
        if ele.tick != 0 || ele.amount != 0 {
            fmt.Println("NewSpeedMeter ele:", ele)
            t.Fail()
        }
    }

    rate := v1.GetRate()
    if rate != 0 {
        fmt.Println("NewSpeedMeter rate:", rate)
        t.Fail()
    }

    var num uint = 10
    time.Sleep(1000 * time.Millisecond)
    v1.Feed(num)

    if v1.preFeedTick <= 0 {
        fmt.Println("NewSpeedMeter preFeedTick:", v1.preFeedTick)
        t.Fail()
    }

    for _, ele := range v1.itemArr {
        if ele.tick == 0 {
            fmt.Println("NewSpeedMeter Feed:", ele)
            t.Fail();
        }
    }
}
