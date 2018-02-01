/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package p2p

import (
	"sync"

	"github.com/aristanetworks/goarista/monotime"
)

type speedMeterSubItem struct {
	tick   uint64
	amount uint
}

//SpeedMeter compute bandwidth
type SpeedMeter struct {
	itemArr     []speedMeterSubItem
	preFeedTick uint32
	step        uint32
	itemsNum    uint32
	mutex       sync.Mutex
}

//NewSpeedMeter create SpeedMeter
func NewSpeedMeter(step uint32, items uint32) (s *SpeedMeter) {
	s = new(SpeedMeter)
	s.step, s.itemsNum = step, items
	s.itemArr = make([]speedMeterSubItem, items)
	return s
}

//Feed cur is milliseconds
func (s *SpeedMeter) Feed(num uint) {
	cur := monotime.Now() / 1000
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.paveToTick(cur)
	curIdx := uint((cur / s.step)) % s.itemsNum
	s.itemArr[curIdx].amount += num
	s.preFeedTick = cur - cur%s.step
}

//GetRate
func (s *SpeedMeter) GetRate() uint {
	cur := monotime.Now() / 1000
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

//paveToTick cur's milliseconds
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
