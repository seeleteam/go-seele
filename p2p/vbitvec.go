/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package p2p

import (
    "fmt"
)

const (
    // MaxBitLength Max bit length
    MaxBitLength uint = 8

    // MaxBitIndex Max bit index size
    MaxBitIndex uint = 7

    // ShiftOffset Left/Right shift size
    ShiftOffset uint = 3
)

// VBitVec bitmap friendly for FEC helper
type VBitVec struct {
    BitLen  uint
    bufLen  uint
    pBuf    []byte
    ExtFlag byte // flag for extra package
}

// Init Initialize VBitVec with bitLen
func (v *VBitVec) Init(_bitLen uint) bool {
    if v.BitLen != 0 {
        panic("VBitVec.Init pacnic")
    }

    v.BitLen, v.bufLen, v.ExtFlag = _bitLen, _bitLen >> ShiftOffset, 0
    if _bitLen & MaxBitIndex != 0 {
        v.bufLen++
    }

    v.pBuf = make([]byte, v.bufLen)
    return true
}

// Attatch Initialize bitmap with memory block
func (v *VBitVec) Attatch(p []byte, _bitLen uint, flag byte) {
    v.BitLen, v.bufLen, v.ExtFlag = _bitLen, _bitLen >> ShiftOffset, flag
    if _bitLen & MaxBitIndex != 0 {
        v.bufLen++
    }

    v.pBuf = p
}

// Detach Clear VBitVec
func (v *VBitVec) Detach() {
    v.BitLen, v.bufLen, v.ExtFlag = 0, 0, 0
    v.pBuf = nil
}

// SetBit Set value by bit index
func (v *VBitVec) SetBit(idx uint, val bool) {
    indx, bitIndx := idx >> ShiftOffset, idx & MaxBitIndex
    if val {
        v.pBuf[indx] |= 1 << (MaxBitIndex - bitIndx)
    } else {
        v.pBuf[indx] &^= 1 << (MaxBitIndex - bitIndx)
    }
}

// GetBit Get bit in pos
func (v *VBitVec) GetBit(idx uint) bool {
    indx, bitIndx := idx >> ShiftOffset, idx & MaxBitIndex
    return ((v.pBuf[indx] >> (MaxBitIndex - bitIndx)) & 1) != 0
}

// GetFlagBit Get bit in pos of extflag
func (v *VBitVec) GetFlagBit(idx uint) bool {
    if idx > MaxBitLength {
        return false
    }

    return ((v.ExtFlag >> (MaxBitIndex - idx)) & 1) != 0
}

// InitPattern Initialize bitmap with pattern
func (v *VBitVec) InitPattern(flag byte, len uint) {
    for idx := uint(0); idx < v.BitLen; idx++ {
        flagIdx := idx % len
        if flag & (1 << (MaxBitIndex - flagIdx)) != 0 {
            v.SetBit(idx, true)
        } else {
            v.SetBit(idx, false)
        }
    }
}

// BitXor Initialize bitmap with c1 ^ c2
func (v *VBitVec) BitXor(c1 *VBitVec, c2 *VBitVec) {
    if (v.BitLen != c1.BitLen) || (c1.BitLen != c2.BitLen) {
        panic("Failed to Xor")
    }

    for i := uint(0); i < v.bufLen; i++ {
        v.pBuf[i] = (c1.pBuf[i]) ^ (c2.pBuf[i])
    }
}

// BitXor1 Xor with c1
func (v *VBitVec) BitXor1(c1 *VBitVec) {
    if v.BitLen != c1.BitLen {
        panic("cannot xor")
    }

    for i := uint(0); i < v.bufLen; i++ {
        v.pBuf[i] = (v.pBuf[i]) ^ (c1.pBuf[i])
    }
}

// has1fecBit Check whether there is only 1 bit set in bitmap
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

// getBitsCnt Calculate bit num
func (v *VBitVec) getBitsCnt(len uint) uint {
    indx := len >> ShiftOffset
    trueCnt := uint(0)
    for i := uint(0); i < indx; i++ {
        for ch := v.pBuf[i]; ch > 0; trueCnt++ {
            ch &= (ch - 1)
        }
    }

    for i := indx << ShiftOffset; i < len; i++ {
        if v.GetBit(i) {
            trueCnt++
        }
    }

    return trueCnt
}

// GetBitmapString for test
func (v *VBitVec) GetBitmapString() (str string) {
    for i := uint(0); i < v.bufLen; i++ {
        str = str + fmt.Sprintf("%02X ", v.pBuf[i])
    }

    return str
}
