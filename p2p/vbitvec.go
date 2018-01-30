package p2p

//"container/list"
import "fmt"

//"fmt"

//VBitVec bitmap friendly for fechelper
type VBitVec struct {
	BitLen  uint
	bufLen  uint
	pBuf    []byte
	ExtFlag byte //flag for extra package
}

//Init init VBitVec, with bitLen
func (v *VBitVec) Init(_bitLen uint) bool {
	if v.BitLen != 0 {
		panic("VBitVec.Init pacnic")
	}

	v.BitLen, v.bufLen = _bitLen, _bitLen>>3
	if _bitLen&7 != 0 {
		v.bufLen++
	}
	v.pBuf = make([]byte, v.bufLen)
	return true
}

//Attatch init bitmap by memory block
func (v *VBitVec) Attatch(p []byte, _bitLen uint, flag byte) {
	v.BitLen, v.bufLen, v.ExtFlag = _bitLen, _bitLen>>3, flag
	if _bitLen&7 != 0 {
		v.bufLen++
	}
	v.pBuf = p
}

//Detach clear VBitVec
func (v *VBitVec) Detach() {
	v.BitLen, v.bufLen, v.ExtFlag = 0, 0, 0
	v.pBuf = nil
}

//SetBit set value by bit index
func (v *VBitVec) SetBit(idx uint, val bool) bool {
	indx, bitIndx := idx>>3, idx&7
	if val {
		v.pBuf[indx] |= 1 << (7 - bitIndx)
	} else {
		v.pBuf[indx] &^= 1 << (7 - bitIndx)
	}
	return val
}

//GetBit get bit in pos
func (v *VBitVec) GetBit(idx uint) bool {
	indx, bitIndx := idx>>3, idx&7
	return ((v.pBuf[indx] >> (7 - bitIndx)) & 1) != 0
}

//GetFlagBit get bit in pos of extflag
func (v *VBitVec) GetFlagBit(idx uint) bool {
	if idx > 8 {
		return false
	}
	return ((v.ExtFlag >> (7 - idx)) & 1) != 0
}

//InitPattern init bitmap by pattern
func (v *VBitVec) InitPattern(flag byte, len uint) {
	for idx := uint(0); idx < v.BitLen; idx++ {
		flagIdx := idx % len
		if flag&(1<<(7-flagIdx)) != 0 {
			v.SetBit(idx, true)
		} else {
			v.SetBit(idx, false)
		}
	}
}

//BitXor init bitmap by c1 ^ c2
func (v *VBitVec) BitXor(c1 *VBitVec, c2 *VBitVec) bool {
	if (v.BitLen != c1.BitLen) || (c1.BitLen != c2.BitLen) {
		panic("cannot xor")
	}
	for i := uint(0); i < v.bufLen; i++ {
		v.pBuf[i] = (c1.pBuf[i]) ^ (c2.pBuf[i])
	}
	return true
}

//BitXor1 compute xor with VBitVec c1
func (v *VBitVec) BitXor1(c1 *VBitVec) bool {
	if v.BitLen != c1.BitLen {
		panic("cannot xor")
	}
	for i := uint(0); i < v.bufLen; i++ {
		v.pBuf[i] = (v.pBuf[i]) ^ (c1.pBuf[i])
	}
	return true
}

//has1fecBit bitmap only has 1 bit, that is set in bitmap,but not in s
func (v *VBitVec) has1fecBit(s *VBitVec) bool {
	diffCnt := uint(0)
	for i := uint(0); i < v.bufLen; i++ {
		ch := (v.pBuf[i]) & (s.pBuf[i])
		ch = ch ^ v.pBuf[i]
		for ; ch > 0; diffCnt++ {
			ch &= (ch - 1)
		}
		if diffCnt >= 2 {
			return false
		}
	}
	return diffCnt == 1
}

//getBitsCnt calculate bit num
func (v *VBitVec) getBitsCnt(len uint) uint {
	indx := len >> 3
	trueCnt := uint(0)
	for i := uint(0); i < indx; i++ {
		for ch := v.pBuf[i]; ch > 0; trueCnt++ {
			ch &= (ch - 1)
		}
	}

	for i := indx << 3; i < len; i++ {
		if v.GetBit(i) {
			trueCnt++
		}
	}
	return trueCnt
}

//GetBitmapString for test
func (v *VBitVec) GetBitmapString() (str string) {
	//fmt.Print("Bitmap:\n\t\t")
	for i := uint(0); i < v.bufLen; i++ {
		str = str + fmt.Sprintf("%02X ", v.pBuf[i])
		//fmt.Print("%X ", (uint)(v.pBuf[i]))
	}
	//fmt.Println("|")
	return str
}
