/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package p2p

import "fmt"

//FECHelper helper class for FEC
type FECHelper struct {
	canVec         [255]*VBitVec
	bitLen, fecLen uint
	idx            uint
}

func (h *FECHelper) createAndInsertByPattern(ch byte, len uint, extFlag byte) {
	p := new(VBitVec)
	p.Init(h.bitLen)
	p.InitPattern(ch, len)
	p.ExtFlag = extFlag
	h.canVec[h.idx] = p
	h.idx++
}

func (h *FECHelper) createOne(arr []int, arrLen int) {
	var flag byte
	for _, v := range arr {
		flag = flag | (1 << (7 - uint(v)))
	}

	fmt.Print(arr)
	fmt.Printf(",%d, %02X\r\n", arrLen, flag)

	p := new(VBitVec)
	p.Init(h.bitLen)
	p.BitXor(h.canVec[arr[0]], h.canVec[arr[1]])
	for idx := 2; idx < arrLen; idx++ {
		p.BitXor1(h.canVec[arr[idx]])
	}

	p.ExtFlag = flag
	h.canVec[h.idx] = p
	h.idx++
}

func (h *FECHelper) selectNum(pre []int, preLen int, arr []int, arrLen int, cnt int) {
	if arrLen == cnt {
		h.createOne(append(pre, arr...), preLen+arrLen)
		return
	}

	if cnt == 1 {
		for _, v := range arr {
			h.createOne(append(pre, v), preLen+1)
		}
		return
	}

	newPreLen, newCnt := preLen+1, cnt-1
	for idx, v := range arr {
		pre1 := make([]int, preLen)
		copy(pre1, pre)
		newPre := append(pre1, v)

		arr1 := make([]int, arrLen)
		copy(arr1, arr)
		newArr := append([]int{}, arr1[idx+1:]...)
		newArrLen := arrLen - idx - 1
		h.selectNum(newPre, newPreLen, newArr, newArrLen, newCnt)
	}
}

//Init initialize helper.
func (h *FECHelper) Init(_bitLen uint) bool {
	if _bitLen%8 != 0 {
		panic("invalid paras, panic...")
	}
	h.bitLen, h.fecLen = _bitLen, 8

	h.createAndInsertByPattern(0x80, 1, 0x80)
	h.createAndInsertByPattern(0x80, 2, 0x40)
	h.createAndInsertByPattern(0x80, 3, 0x20)
	h.createAndInsertByPattern(0x40, 3, 0x10)
	h.createAndInsertByPattern(0xdb, 7, 0x08)
	h.createAndInsertByPattern(0xba, 7, 0x04)
	h.createAndInsertByPattern(0x6e, 7, 0x02)
	h.createAndInsertByPattern(0xcd, 7, 0x01)
	////modify
	h.canVec[0].SetBit(14, false)
	h.canVec[0].SetBit(12, false)

	org := []int{0, 1, 2, 3, 4, 5, 6, 7}
	pre := []int{}
	for i := 2; i <= 8; i++ {
		h.selectNum(pre, 0, org, 8, i)
	}
	return true
}
