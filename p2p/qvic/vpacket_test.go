/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package qvic

import (
	"bytes"
	"fmt"
	"testing"
)

func Test_MarshalData(t *testing.T) {
	v1 := createVPacket("Golang")
	v1.MarshalData()

	// meta info
	if v1.seq != 101 || v1.packType != 2 || v1.fecIdx != 4 || v1.magic != 120 ||
		v1.lastSeqSendTick != 0 || v1.createTick != 0 || v1.dataLen != 6 {
		fmt.Println("Test_MarshalData:", v1.seq, v1.packType, v1.fecIdx, v1.magic, v1.lastSeqSendTick, v1.createTick, v1.dataLen)
		t.Fail()
	}

	// data
	if !bytes.Equal(v1.data, []byte("Golang")) {
		fmt.Println("Test_MarshalData data:", v1.data)
		t.Fail()
	}

	// data is included in dataNet
	if !bytes.Equal(v1.dataNet[13:19], v1.data) {
		fmt.Println("Test_MarshalData dataNet:", v1.dataNet[13:19], v1.data)
		t.Fail()
	}

	// rest of dataNet must be 0
	zeroArray := [1500]byte{}
	if !bytes.Equal(v1.dataNet[19:], zeroArray[19:]) {
		fmt.Println("Test_MarshalData:", v1.dataNet[19:])
		t.Fail()
	}
}

func Test_ParseData(t *testing.T) {
	v1 := createVPacket("Golang")
	v1.MarshalData()

	v2 := createVPacket("should be the same as Golang")
	v2.ParseData(v1.dataNet[:])

	// meta info
	if v1.seq != v2.seq || v1.packType != v2.packType || v1.fecIdx != v2.fecIdx || v1.magic != v2.magic ||
		v1.lastSeqSendTick != v2.lastSeqSendTick || v1.createTick != v2.createTick {
		fmt.Println("Test_ParseData v1:", v1.seq, v1.packType, v1.fecIdx, v1.magic, v1.lastSeqSendTick, v1.createTick)
		fmt.Println("Test_ParseData v2:", v2.seq, v2.packType, v2.fecIdx, v2.magic, v2.lastSeqSendTick, v2.createTick)
		t.Fail()
	}

	// data
	if !bytes.Equal(v1.data[:6], v2.data[:6]) {
		fmt.Println("Test_ParseData data:", v1.data, v2.data)
		t.Fail()
	}

	// dataNet
	if !bytes.Equal(v1.dataNet[:], v2.dataNet[:]) {
		fmt.Println("Test_ParseData dataNet:", v1.dataNet, v2.dataNet)
		t.Fail()
	}
}

func createVPacket(data string) (v *VPacket) {
	v1 := new(VPacket)
	v1.seq, v1.packType, v1.fecIdx, v1.magic = 101, 2, 4, 120
	v1.data = []byte(data)
	v1.dataLen = len(v1.data)
	return v1
}
