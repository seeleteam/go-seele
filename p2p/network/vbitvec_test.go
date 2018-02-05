/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"bytes"
	"fmt"
	"testing"
)

func Test_VBitVec_check(t *testing.T) {
	v1 := new(VBitVec)
	v1.Init(64)
	v2 := new(VBitVec)
	v2.Init(64)
	v1.InitPattern(0x80, 1)
	v2.InitPattern(0x80, 1)
	v2.SetBit(0, false)
	v2.SetBit(1, false)
	ret := v1.has1fecBit(v2)

	if ret != false {
		fmt.Println("VBitVec_check ret=", ret)
		t.Fail()
	}
}

func Test_Init(t *testing.T) {
	v1 := new(VBitVec)
	v1.Init(64)

	if v1.BitLen != 64 || v1.bufLen != 8 || v1.ExtFlag != 0 {
		fmt.Println("Init", v1.BitLen, v1.bufLen, v1.ExtFlag)
		t.Fail()
	}

	// pBuf must be 0
	zeroArray := [8]byte{}
	if !bytes.Equal(v1.pBuf[:], zeroArray[:]) {
		fmt.Println("Init pBuf:", v1.pBuf)
		t.Fail()
	}
}

func Test_Attatch(t *testing.T) {
	v1 := new(VBitVec)
	v1.Init(64)

	p := []byte{1, 2, 3}
	var flag byte

	for bitLen := uint(1); bitLen < 64; bitLen++ {
		v1.Attatch(p, bitLen, flag)

		if v1.BitLen != bitLen || v1.ExtFlag != flag || !bytes.Equal(v1.pBuf[:], p[:]) {
			fmt.Println("Attatch", v1.BitLen, v1.bufLen, v1.ExtFlag)
			t.Fail()
		}

		// Checks the bufLen
		var bufLen = bitLen / MaxBitLength
		if bufLen*MaxBitLength < bitLen {
			bufLen++
		}

		if v1.bufLen != bufLen {
			fmt.Println("Attatch bufLen:", v1.bufLen, bufLen)
			t.Fail()
		}
	}
}

func Test_Detach(t *testing.T) {
	v1 := new(VBitVec)
	v1.Init(64)
	v1.Detach()

	if v1.BitLen != 0 || v1.bufLen != 0 || v1.ExtFlag != 0 {
		fmt.Println("Detach", v1.BitLen, v1.bufLen, v1.ExtFlag)
		t.Fail()
	}

	if v1.pBuf != nil {
		fmt.Println("Detach pBuf:", v1.pBuf)
		t.Fail()
	}
}

func Test_SetAndGetBit(t *testing.T) {
	v1 := new(VBitVec)
	v1.Init(64)

	// 1
	v1.SetBit(1, true)
	p := []byte{64, 0, 0, 0, 0, 0, 0, 0}
	if !bytes.Equal(v1.pBuf, p) {
		fmt.Println("Detach pBuf:", v1.pBuf)
		t.Fail()
	}

	if !v1.GetBit(1) {
		t.Fail()
	}

	// 2
	v1.SetBit(2, true)
	p = []byte{96, 0, 0, 0, 0, 0, 0, 0}
	if !bytes.Equal(v1.pBuf, p) {
		fmt.Println("Detach pBuf:", v1.pBuf)
		t.Fail()
	}

	if !v1.GetBit(2) {
		t.Fail()
	}

	// 3
	v1.SetBit(3, false)
	p = []byte{96, 0, 0, 0, 0, 0, 0, 0}
	if !bytes.Equal(v1.pBuf, p) {
		fmt.Println("Detach pBuf:", v1.pBuf)
		t.Fail()
	}

	if v1.GetBit(3) {
		t.Fail()
	}
}

func Test_BitXor(t *testing.T) {
	v1 := new(VBitVec)
	v1.Init(64)

	v2 := new(VBitVec)
	v2.Init(64)

	v3 := new(VBitVec)
	v3.Init(64)
	v3.SetBit(1, true)

	v1.BitXor(v2, v3)
	if !bytes.Equal(v1.pBuf, v3.pBuf) {
		fmt.Println("Detach pBuf:", v1.pBuf, v3.pBuf)
		t.Fail()
	}

	v4 := new(VBitVec)
	v4.Init(64)
	v4.SetBit(2, true)
	v2.BitXor1(v4)
	if !bytes.Equal(v2.pBuf, v4.pBuf) {
		fmt.Println("Detach pBuf:", v2.pBuf, v4.pBuf)
		t.Fail()
	}
}
