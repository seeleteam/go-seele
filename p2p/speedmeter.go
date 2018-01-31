package p2p

import (
	"fmt"
)

type speedMeterSubItem struct {
	tick   uint32
	amount uint32
}

//SpeedMeter compute bandwidth
type SpeedMeter struct {
	itemArr     []speedMeterSubItem
	preFeedTick uint32
	step        uint32
	itemsNum    uint32
}

//NewSpeedMeter create SpeedMeter
func NewSpeedMeter(step uint32, items uint32) (s *SpeedMeter) {
	s = new(SpeedMeter)
	s.step, s.itemsNum = step, items
	s.itemArr = make([]speedMeterSubItem, items)
	return s
}
