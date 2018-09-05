/**
*  Package p2p
*  @copyright defined in go-seele/LICENSE
 */

package qvic

import (
	"encoding/binary"
)

const (
	// VPacketHeadLen head length in net-package
	VPacketHeadLen  int = 13
	PackTypeControl int = 8
	PackTypeChannel int = 7
	PackTypeData    int = 6
	PackTypeFEC     int = 5
)

// VPacket package for QVIC
type VPacket struct {
	magic           uint32
	packType        byte
	fecIdx          int
	seq             uint32
	lastSeqSendTick uint16
	createTick      uint16
	data            []byte
	dataLen         int
	isRecovered     int // 1: ok; 0: others. for client, 1: recved ack; for server, 1: ack sent.
	isSendedToPeer  bool
	sendTimes       int
	myPos           int
	dataNet         [1500]byte
}

// MarshalData pack data to dataNet
func (v *VPacket) MarshalData() {
	b := v.dataNet[0:]
	binary.BigEndian.PutUint32(b[0:4], v.magic)
	b[4] = (v.packType << 4) | byte(v.fecIdx)
	//binary.BigEndian.PutUint16(b[5:7], v.crc)
	binary.BigEndian.PutUint32(b[5:9], v.seq)
	binary.BigEndian.PutUint16(b[9:11], v.lastSeqSendTick)
	binary.BigEndian.PutUint16(b[11:13], v.createTick)
	copy(b[VPacketHeadLen:], v.data)
}

// ParseData parse data from dataNet. packData contains udp-package recved from net
func (v *VPacket) ParseData(packData []byte) {
	b := v.dataNet[0:]
	copy(b, packData)
	v.magic = binary.BigEndian.Uint32(b[:4])
	v.packType, v.fecIdx = b[4]>>4, int(b[4]&0x0f)
	v.seq = binary.BigEndian.Uint32(b[5:9])
	v.lastSeqSendTick = binary.BigEndian.Uint16(b[9:11])
	v.createTick = binary.BigEndian.Uint16(b[11:13])
	v.dataLen = len(packData) - VPacketHeadLen
	v.data = make([]byte, v.dataLen)
	copy(v.data, b[VPacketHeadLen:])
}

// FECInfo record packages info of a bundle
type FECInfo struct {
	seq        uint32
	fecPackets [8]*VPacket
	fecFlag    byte
}

// NewFECInfo create new FECInfo class
func NewFECInfo() (f *FECInfo) {
	f = new(FECInfo)
	return f
}

// Clear reset FECInfo
func (v *FECInfo) Clear() {
	if v.fecFlag == 0 {
		return
	}
	for i := 0; i < 8; i++ {
		if v.fecPackets[i] != nil {
			v.fecPackets[i] = nil
		}
	}
	v.fecFlag = 0
}

// VPacketItem package info for RecverMgr
type VPacketItem struct {
	pFECInfo      *FECInfo
	p             *VPacket
	fecCreateTick uint16
}
