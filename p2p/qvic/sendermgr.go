/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package qvic

import (
	"encoding/binary"
	"net"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/log"
)

// SenderMgr manages QConn's sender logic.
type SenderMgr struct {
	lock  sync.Mutex
	udpfd *net.UDPConn

	loopWG sync.WaitGroup
	log    *log.SeeleLog

	senderSlice          []*VPacket
	qconn                *QConn
	winSize              uint32
	fecHelper            *FECHelper
	bundleNum, fecExtLen int
	curSeq               uint32
	peerCurSeq           uint32
	peerWinStartSeq      uint32
}

// NewSenderMgr creates SenderMgr object.
func NewSenderMgr(qc *QConn) *SenderMgr {
	s := &SenderMgr{
		fecHelper:       qc.fecHelper,
		winSize:         DefaultWinSize,
		bundleNum:       DefaultFECBundle,
		fecExtLen:       DefaultFECLen,
		qconn:           qc,
		udpfd:           qc.udpFD,
		curSeq:          qc.sendStartSeq,
		peerCurSeq:      qc.sendStartSeq,
		peerWinStartSeq: qc.sendStartSeq,
		log:             qc.log,
	}

	s.senderSlice = make([]*VPacket, 0, s.winSize)
	s.loopWG.Add(1)
	go s.sendHeartBeat()
	s.log.Info("QVIC NewSenderMgr started!")
	return s
}

func (sm *SenderMgr) sendHeartBeat() {
	defer sm.loopWG.Done()
	ticker := time.NewTicker(1 * time.Second)
needQuit:
	for {
		select {
		case <-ticker.C:
			sm.doSendHB()
		case <-sm.qconn.quit:
			break needQuit
		}
	}
	sm.log.Debug("sendHeartBeat routine out!")
}

func (sm *SenderMgr) doSendHB() {
	var b [6]byte
	binary.BigEndian.PutUint32(b[:4], sm.qconn.magic)
	b[4] = byte(PackTypeChannel << 4)
	b[5] = msgChannelHeartBeat
	sLen, err := sm.udpfd.WriteTo(b[0:6], sm.qconn.RemoteAddr())
	sm.log.Info("senderMgr doSendHB. sLen=%d err=%s", sLen, err)
}

// onBitmap handles bitmap message
func (sm *SenderMgr) onBitmap(data []byte, curTick uint32) {
	msgCurSeq := binary.BigEndian.Uint32(data[6:10])
	msgMaxSeq := binary.BigEndian.Uint32(data[10:14])
	msgLastArrived_SendTick := binary.BigEndian.Uint16(data[14:16])
	msgLastArrived_DurationTick := binary.BigEndian.Uint16(data[16:18])
	msgStartPackNo := binary.BigEndian.Uint32(data[18:22])
	msgEndPackNo := binary.BigEndian.Uint32(data[22:26])
	msgWinStartPackNo := binary.BigEndian.Uint32(data[26:30])
	msgBitmap := data[30:]

	sm.lock.Lock()
	defer sm.lock.Unlock()
	if len(sm.senderSlice) == 0 {
		return
	}
	sm.log.Debug("onBitmap msgCurSeq=%d msgMaxSeq=%d bitmapLen=%d msgWinStartPackNo=%d curSeq=%d len(packSlice)=%d",
		msgCurSeq, msgMaxSeq, len(data)-30, msgWinStartPackNo, sm.curSeq, len(sm.senderSlice))
	sm.peerWinStartSeq = msgWinStartPackNo
	if !sm.qconn.isEqAndBig_32(sm.peerCurSeq, msgCurSeq) {
		localStartSeq := sm.senderSlice[0].seq
		for curSeq := sm.peerCurSeq; curSeq != msgCurSeq; curSeq++ {
			pack := sm.senderSlice[curSeq-localStartSeq]
			if pack.isSendedToPeer {
				continue
			}
			pack.isSendedToPeer = true
		}

		sm.peerCurSeq = msgCurSeq
		sm.log.Debug("onBitmap m_peerCurSeq=%d", msgCurSeq)
		sm.doDel()
	}

	localStartSeq := sm.senderSlice[0].seq
	pack := sm.senderSlice[len(sm.senderSlice)-1]
	if msgMaxSeq != pack.seq {
		if sm.qconn.isEqAndBig_16(msgLastArrived_SendTick, pack.lastSeqSendTick+sm.qconn.jitter-msgLastArrived_DurationTick) {
			myMaxSeq := pack.seq
			for curSeq := msgMaxSeq + 1; curSeq != (myMaxSeq + 1); curSeq++ {
				if !sm.qconn.isEqAndBig_32(curSeq, sm.peerCurSeq) {
					continue
				}
				pack = sm.senderSlice[curSeq-localStartSeq]
				if sm.qconn.isEqAndBig_16(msgLastArrived_SendTick, pack.lastSeqSendTick+sm.qconn.jitter-msgLastArrived_DurationTick) {
					pack.lastSeqSendTick = uint16(curTick)

					pack.sendTimes++
					pack.MarshalData()
					sm.udpfd.WriteTo(pack.dataNet[0:pack.dataLen+VPacketHeadLen], sm.qconn.RemoteAddr())
				}
			}
		}
	}

	if msgStartPackNo == msgEndPackNo {
		return
	}

	if sm.qconn.isEqAndBig_32(msgEndPackNo, msgMaxSeq) {
		msgEndPackNo = msgMaxSeq
	}

	for curSeq := msgStartPackNo; curSeq != msgEndPackNo; curSeq++ {
		if !sm.qconn.isEqAndBig_32(curSeq, sm.peerCurSeq) {
			continue
		}

		pack = sm.senderSlice[curSeq-localStartSeq]
		idx := curSeq - msgStartPackNo
		indx, bitIndx := idx>>3, idx&7
		value := 1 << (7 - bitIndx)
		if msgBitmap[indx]&byte(value) != 0 {
			if pack.isSendedToPeer {
				continue
			}
			pack.isSendedToPeer = true
			continue
		}

		if pack.isSendedToPeer {
			continue
		}

		if sm.qconn.isEqAndBig_16(msgLastArrived_SendTick, pack.lastSeqSendTick+sm.qconn.jitter-msgLastArrived_DurationTick) {
			pack.lastSeqSendTick = uint16(curTick)

			pack.sendTimes++
			pack.MarshalData()
			sm.udpfd.WriteTo(pack.dataNet[0:pack.dataLen+VPacketHeadLen], sm.qconn.RemoteAddr())
		}
	}
}

func (sm *SenderMgr) min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func (sm *SenderMgr) doDel() {
	if len(sm.senderSlice) < 256 {
		return
	}

	localStartSeq := sm.senderSlice[0].seq
	toDel := sm.min(int(sm.peerCurSeq-localStartSeq), len(sm.senderSlice)-256)
	if toDel == 0 {
		return
	}
	sm.log.Debug("qvic sendermgr doDel %d packets", toDel)
	sm.senderSlice = sm.senderSlice[toDel:]
}

func (sm *SenderMgr) trySendPacket(p *VPacket, curTick uint32) bool {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	if sm.qconn.isEqAndBig_32(sm.curSeq, sm.peerWinStartSeq+sm.winSize-256) {
		return false
	}

	p.lastSeqSendTick = p.createTick
	p.seq, p.sendTimes = sm.curSeq, 1
	sm.curSeq = sm.curSeq + 1
	p.MarshalData()
	sm.sendPacket(p, curTick)
	return true
}

func (sm *SenderMgr) sendPacket(p *VPacket, curTick uint32) {
	sm.senderSlice = append(sm.senderSlice, p)
	sm.udpfd.WriteTo(p.dataNet[0:p.dataLen+VPacketHeadLen], sm.qconn.RemoteAddr())
	sm.trySendFecPackets(p, curTick)
}

func (sm *SenderMgr) tryRelaySendPackets() {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	need := sm.bundleNum - int(sm.curSeq%uint32(sm.bundleNum))
	if need == 0 {
		return
	}
	begin := sm.curSeq
	go func() {
		time.Sleep(10 * time.Millisecond)
		sm.log.Debug("tryRelaySendPackets func")
		sm.lock.Lock()
		defer sm.lock.Unlock()
		if begin != sm.curSeq {
			return
		}
		curTick := time.Now().UnixNano() / (1000 * 1000)
		nullData := make([]byte, 2)
		for i := 0; i < need; i++ {
			p := new(VPacket)
			p.packType, p.magic, p.createTick = byte(PackTypeData), sm.qconn.magic, uint16(curTick)
			p.data, p.dataLen = nullData, 2
			p.lastSeqSendTick = p.createTick
			p.seq, p.sendTimes = sm.curSeq, 1
			sm.curSeq = sm.curSeq + 1

			p.MarshalData()
			sm.sendPacket(p, uint32(curTick))
		}
	}()
	return
}

// trySendFecPackets sends fec packets if necessary
func (sm *SenderMgr) trySendFecPackets(p *VPacket, curTick uint32) {
	if (p.seq%uint32(sm.bundleNum)) != (uint32(sm.bundleNum)-1) ||
		len(sm.senderSlice) < sm.bundleNum*2 {
		return
	}

	pack := new(VPacket)
	pack.seq, pack.packType, pack.magic, pack.createTick = p.seq, byte(PackTypeFEC), p.magic, uint16(curTick)
	startIdx := len(sm.senderSlice) - sm.bundleNum
	for idx := 0; idx < sm.fecExtLen; idx++ {
		pBitVec := sm.qconn.fecHelper.canVec[idx]
		pack.fecIdx = idx
		pack.data = make([]byte, sm.qconn.packDataSize)
		packDataLen := 0
		for i := 0; i < sm.bundleNum; i++ {
			if !pBitVec.GetBit(uint(i)) {
				continue
			}

			p1 := sm.senderSlice[startIdx+i]
			for j := 0; j < p1.dataLen; j++ {
				pack.data[j] = pack.data[j] ^ p1.data[j]
			}

			if packDataLen < p1.dataLen {
				packDataLen = p1.dataLen
			}
		}
		pack.dataLen = packDataLen
		pack.MarshalData()
		sm.udpfd.WriteTo(pack.dataNet[0:packDataLen+VPacketHeadLen], sm.qconn.RemoteAddr())
	}
}

func (sm *SenderMgr) close() {
	sm.loopWG.Wait()
}
