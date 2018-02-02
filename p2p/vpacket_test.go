/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */
package p2p

import (
	"fmt"
	"testing"
)

func Test_VTunPacket_1(t *testing.T) {
	v1 := new(VPacket)
	v1.seq, v1.packType, v1.fecIdx, v1.crc, v1.magic = 101, 2, 4, 110, 120
	v1.data = []byte("Golang")
	v1.dataLen = uint(len(v1.data))

	v1.MarshalData()
	v2 := new(VPacket)
	slice1 := v1.dataNet[:v1.dataLen+VPacketHeadLen]

	v2.ParseData(slice1)
	fmt.Println("Test_VTunPacket_1 1", v1.seq, v1.packType, v1.fecIdx, v1.crc, v1.magic, v1.data, v1.dataLen)
	fmt.Println("Test_VTunPacket_1 2", v2.seq, v2.packType, v2.fecIdx, v2.crc, v2.magic, v2.data, v2.dataLen)
}
