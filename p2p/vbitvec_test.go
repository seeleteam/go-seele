package p2p

import (
	"fmt"
	"testing"
)

func Test_abc(t *testing.T) {
	fmt.Println("abc")
	t.Logf("abc from t")
	v := new(VBitVec)
	v.Init(32)
	//v.LogBitmap()
}

func Test_has1fecBit(t *testing.T) {
	v1 := new(VBitVec)
	v1.Init(64)
	v2 := new(VBitVec)
	v2.Init(64)
	v1.InitPattern(0x80, 1)
	v2.InitPattern(0x80, 1)
	v2.SetBit(0, false)
	v2.SetBit(1, false)
	ret := v1.has1fecBit(v2)
	fmt.Println("Test_has1fecBit ret=", ret)
	//return ret
}
