/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package qvic

import (
	"sync"

	"github.com/aristanetworks/goarista/monotime"
)

const (
	// MilliInSec milliseconds in one second
	MilliInSec uint64 = 1000
)

// speedMeterSubItem records amount in a step
type speedMeterSubItem struct {
	tick   uint64
	amount uint
}

// SpeedMeter computes bandwidth
type SpeedMeter struct {
	itemArr     []speedMeterSubItem
	preFeedTick uint64 // last feed tick
	step        uint64 //
	itemsNum    uint   // step num in a period
	mutex       sync.Mutex
}

// NewSpeedMeter creates SpeedMeter.
// step should be ms. for example: step=100 items=10; step=50 items=20.
// step * items = a period.
func NewSpeedMeter(step uint64, items uint) (s *SpeedMeter) {
	s = new(SpeedMeter)
	s.step, s.itemsNum = step, items
	s.itemArr = make([]speedMeterSubItem, items)
	return s
}

// Feed called when bytes received from network
func (s *SpeedMeter) Feed(num uint) {
	cur := monotime.Now() / MilliInSec
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.paveToTick(cur)
	curIdx := uint((cur / s.step)) % s.itemsNum
	s.itemArr[curIdx].amount += num
	s.preFeedTick = cur - cur%s.step
}

// GetRate gets rate
func (s *SpeedMeter) GetRate() uint {
	cur := monotime.Now() / MilliInSec
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.paveToTick(cur)
	curIdx := uint((cur / s.step)) % s.itemsNum
	curAmount := s.itemArr[curIdx].amount

	var firstAmount uint
	if curIdx == (s.itemsNum - 1) {
		firstAmount = s.itemArr[0].amount
	} else {
		firstAmount = s.itemArr[curIdx+1].amount
	}

	return curAmount - firstAmount
}

// paveToTick cur's milliseconds
func (s *SpeedMeter) paveToTick(cur uint64) {
	preIdx := uint((s.preFeedTick / s.step)) % s.itemsNum
	preAmount := s.itemArr[preIdx].amount
	cur = cur - cur%s.step
	for i := uint(0); i < s.itemsNum; i++ {
		tick := cur - uint64(i)*s.step
		if tick == s.preFeedTick {
			break
		}

		idx := uint((tick / s.step)) % s.itemsNum
		s.itemArr[idx].tick, s.itemArr[idx].amount = tick, preAmount
	}
}
