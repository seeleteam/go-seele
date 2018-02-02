/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package p2p

//FECHelper helper class for FEC
type FECHelper struct {
	canVec         []*VBitVec //candidate vector
	bitLen, fecLen uint
}

//createAndInsertByPattern create VBitVec by pattern, and append to canVec
func (h *FECHelper) createAndInsertByPattern(ch byte, len uint, extFlag byte) {
	p := new(VBitVec)
	p.Init(h.bitLen)
	p.InitPattern(ch, len)
	p.ExtFlag = extFlag
	h.canVec = append(h.canVec, p)
}

//createOne create one VBitVec, and append to canVec
func (h *FECHelper) createOne(arr []int, arrLen int) {
	var flag byte
	for _, v := range arr {
		flag = flag | (1 << (7 - uint(v)))
	}

	p := new(VBitVec)
	p.Init(h.bitLen)
	p.BitXor(h.canVec[arr[0]], h.canVec[arr[1]])
	for idx := 2; idx < arrLen; idx++ {
		p.BitXor1(h.canVec[arr[idx]])
	}

	p.ExtFlag = flag
	h.canVec = append(h.canVec, p)
}

//selectNum create all VBitVec
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
	h.canVec = make([]*VBitVec, 0, 255)
	h.bitLen, h.fecLen = _bitLen, 8

	h.createAndInsertByPattern(0x80, 1, 0x80)
	h.createAndInsertByPattern(0x80, 2, 0x40)
	h.createAndInsertByPattern(0x80, 3, 0x20)
	h.createAndInsertByPattern(0x40, 3, 0x10)
	h.createAndInsertByPattern(0xdb, 7, 0x08)
	h.createAndInsertByPattern(0xba, 7, 0x04)
	h.createAndInsertByPattern(0x6e, 7, 0x02)
	h.createAndInsertByPattern(0xcd, 7, 0x01)
	h.canVec[0].SetBit(14, false)
	h.canVec[0].SetBit(12, false)

	org := []int{0, 1, 2, 3, 4, 5, 6, 7}
	pre := []int{}
	for i := 2; i <= 8; i++ {
		h.selectNum(pre, 0, org, 8, i)
	}
	return true
}

//GetRecoverInfo get VBitVec that can recover
func (h *FECHelper) GetRecoverInfo(pS *VBitVec) *VBitVec {
	if pS.ExtFlag == 0 {
		return nil
	}
	var pCur *VBitVec
	var curBits uint
	for _, p := range h.canVec {
		if p.ExtFlag|pS.ExtFlag != pS.ExtFlag {
			continue
		}
		if p.has1fecBit(pS) {
			if pCur == nil {
				pCur, curBits = p, p.getBitsCnt(pS.BitLen)
				continue
			}
			cnt := p.getBitsCnt(pS.BitLen)
			if cnt < curBits {
				pCur = p
				curBits = cnt
			}
		}
	}
	return pCur
}
