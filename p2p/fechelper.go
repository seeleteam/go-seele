package p2p

import (
	"fmt"

	"github.com/aristanetworks/goarista/monotime"
)

//CFECHelper helper class for fec
type CFECHelper struct {
	canVec         [255]*VBitVec
	bitLen, fecLen uint
}

//Init initialize helper.
func (h *CFECHelper) Init(_bitLen uint, _fecLen uint) bool {
	t1 := monotime.Now()
	fmt.Println("sss", t1)
	if _bitLen%8 != 0 || _fecLen > 8 {
		panic("invalid paras, panic...")
	}
	h.bitLen, h.fecLen = _bitLen, _fecLen

	idx := uint(0)
	p := new(VBitVec)
	p.Init(_bitLen)
	p.InitPattern(0x80, 1)
	p.ExtFlag = 0x80
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.InitPattern(0x80, 2)
	p.ExtFlag = 0x40
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.InitPattern(0x80, 3)
	p.ExtFlag = 0x20
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.InitPattern(0x40, 3)
	p.ExtFlag = 0x10
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.InitPattern(0xdb, 7)
	p.ExtFlag = 0x08
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.InitPattern(0xba, 7)
	p.ExtFlag = 0x04
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.InitPattern(0x6e, 7)
	p.ExtFlag = 0x02
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.InitPattern(0xcd, 7)
	p.ExtFlag = 0x01
	h.canVec[idx] = p
	idx++
	//////
	////modify
	h.canVec[0].SetBit(14, false)
	h.canVec[0].SetBit(12, false)
	/////
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.ExtFlag = 0xC0
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.ExtFlag = 0xA0
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[3])
	p.ExtFlag = 0x90
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[4])
	p.ExtFlag = 0x88
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[5])
	p.ExtFlag = 0x84
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[6])
	p.ExtFlag = 0x82
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[7])
	p.ExtFlag = 0x81
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.ExtFlag = 0x60
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[3])
	p.ExtFlag = 0x50
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[4])
	p.ExtFlag = 0x48
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[5])
	p.ExtFlag = 0x44
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[6])
	p.ExtFlag = 0x42
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[7])
	p.ExtFlag = 0x41
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[3])
	p.ExtFlag = 0x30
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[4])
	p.ExtFlag = 0x28
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[5])
	p.ExtFlag = 0x24
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[6])
	p.ExtFlag = 0x22
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[7])
	p.ExtFlag = 0x21
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[3], h.canVec[4])
	p.ExtFlag = 0x18
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[3], h.canVec[5])
	p.ExtFlag = 0x14
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[3], h.canVec[6])
	p.ExtFlag = 0x12
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[3], h.canVec[7])
	p.ExtFlag = 0x11
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[4], h.canVec[5])
	p.ExtFlag = 0x0C
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[4], h.canVec[6])
	p.ExtFlag = 0x0A
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[4], h.canVec[7])
	p.ExtFlag = 0x09
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[5], h.canVec[6])
	p.ExtFlag = 0x06
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[5], h.canVec[7])
	p.ExtFlag = 0x05
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[6], h.canVec[7])
	p.ExtFlag = 0x03
	h.canVec[idx] = p
	idx++

	///3个56 item
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.ExtFlag = 0xE0
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[3])
	p.ExtFlag = 0xD0
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[4])
	p.ExtFlag = 0xC8
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0xC4
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xC2
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xC1
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.ExtFlag = 0xB0
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.ExtFlag = 0xA8
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0xA4
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xA2
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xA1
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.ExtFlag = 0x98
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0x94
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[3])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x92
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[3])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x91
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0x8C
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x8A
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[4])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x89
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x86
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x85
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x83
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.ExtFlag = 0x70
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.ExtFlag = 0x68
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0x64
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x62
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x61
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.ExtFlag = 0x58
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0x54
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[3])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x52
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[3])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x51
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0x4C
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x4A
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[4])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x49
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x46
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x45
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x43
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.ExtFlag = 0x38
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0x34
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[3])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x32
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[3])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x31
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0x2C
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x2A
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[4])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x29
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x26
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x25
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x23
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[3], h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0x1C
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[3], h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x1A
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[3], h.canVec[4])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x19
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[3], h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x16
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[3], h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x15
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[3], h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x13
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[4], h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x0E
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[4], h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x0D
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[4], h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x0B
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[5], h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x07
	h.canVec[idx] = p
	idx++

	////4个 70items
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.ExtFlag = 0xF0
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.ExtFlag = 0xE8
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0xE4
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xE2
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xE1
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.ExtFlag = 0xD8
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0xD4
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xD2
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xD1
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0xCC
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xCA
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xC9
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xC6
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xC5
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xC3
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.ExtFlag = 0xB8
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0xB4
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xB2
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xB1
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0xAC
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xAA
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xA9
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xA6
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xA5
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xA3
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0x9C
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x9A
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x99
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x96
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x95
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[3])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x93
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x8E
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x8D
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x8B
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x87
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.ExtFlag = 0x78
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0x74
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x72
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x71
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0x6C
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x6A
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x69
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x66
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x65
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x63
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0x5C
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x5A
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x59
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x56
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x55
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[3])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x53
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x4E
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x4D
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x4B
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x47
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0x3C
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x3A
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x39
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x36
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x35
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[3])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x33
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x2E
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x2D
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x2B
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x27
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[3], h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x1E
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[3], h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x1D
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[3], h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x1B
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[3], h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x17
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[4], h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x0F
	h.canVec[idx] = p
	idx++

	//5个 56items
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.ExtFlag = 0xF8
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0xF4
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xF2
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xF1
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0xEC
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xEA
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xE9
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xE6
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xE5
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xE3
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0xDC
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xDA
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xD9
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xD6
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xD5
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xD3
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xCE
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xCD
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xCB
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xC7
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0xBC
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xBA
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xB9
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xB6
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xB5
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xB3
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xAE
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xAD
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xAB
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xA7
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x9E
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x9D
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x9B
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x97
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x8F
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0x7C
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x7A
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x79
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x76
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x75
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x73
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x6E
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x6D
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x6B
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x67
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x5E
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x5D
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x5B
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x57
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x4F
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x3E
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x3D
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x3B
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x37
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x2F
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[3], h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x1F
	h.canVec[idx] = p
	idx++

	//6个。28个
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.ExtFlag = 0xFC
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xFA
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xF9
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xF6
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xF5
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xF3
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xEE
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xED
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xEB
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xE7
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xDE
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xDD
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xDB
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xD7
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xCF
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xBE
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xBD
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xBB
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xB7
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xAF
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x9F
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0x7E
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x7D
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x7B
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x77
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x6F
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x5F
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[2], h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x3F
	h.canVec[idx] = p
	idx++

	///7个。8 items
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xFE
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xFD
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xFB
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xF7
	h.canVec[idx] = p
	idx++

	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xEF
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xDF
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0xBF
	h.canVec[idx] = p
	idx++
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[1], h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[7])
	p.ExtFlag = 0x7F
	h.canVec[idx] = p
	idx++

	///8个。 1items
	p = new(VBitVec)
	p.Init(_bitLen)
	p.BitXor(h.canVec[0], h.canVec[1])
	p.BitXor1(h.canVec[2])
	p.BitXor1(h.canVec[3])
	p.BitXor1(h.canVec[4])
	p.BitXor1(h.canVec[5])
	p.BitXor1(h.canVec[6])
	p.BitXor1(h.canVec[6])
	p.ExtFlag = 0xFF
	h.canVec[idx] = p
	idx++

	return true
}

/*
func main(){
	//var n int = 5;
	 v1 := new(VBitVec)
	 ret := v1.Init(64)
	fmt.Println(v1.bitLen, ret)
}
*/
