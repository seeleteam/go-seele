/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package qvic

import (
	"encoding/binary"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/seeleteam/go-seele/log"
)

// ReceiverMgr manages QConn's receiver logic.
type ReceiverMgr struct {
	lock   sync.Mutex
	log    *log.SeeleLog
	loopWG sync.WaitGroup

	qconn                      *QConn
	udpfd                      *net.UDPConn
	winSize                    uint32
	fecHelper                  *FECHelper
	bundleNum, fecExtLen       int
	startSeq                   uint32 // first seqno of packItems
	curReadSeq                 uint32 // current seqno for read
	curSeq                     uint32 // smallest seqno that has not been received.
	maxSeq                     uint32 //max seqno received by now
	maxSeqNoCreateTick         uint16
	maxSeqNoLocalTick          uint32
	least_peer_lastSeqSendTick uint16
	least_local_RecvedTick     uint32
	bHasPacketRecved           bool
	lastHBTime                 time.Time

	packItems []VPacketItem
	bitmap    []byte
}

// NewReceiverMgr creates ReceiverMgr object.
func NewReceiverMgr(qc *QConn) *ReceiverMgr {
	r := &ReceiverMgr{
		fecHelper:        qc.fecHelper,
		winSize:          DefaultWinSize,
		bundleNum:        DefaultFECBundle,
		fecExtLen:        DefaultFECLen,
		qconn:            qc,
		udpfd:            qc.udpFD,
		curSeq:           qc.peerStartSeq,
		curReadSeq:       qc.peerStartSeq,
		startSeq:         qc.peerStartSeq,
		maxSeq:           qc.peerStartSeq,
		bHasPacketRecved: false,
		lastHBTime:       time.Now(),
		packItems:        make([]VPacketItem, DefaultWinSize),
		bitmap:           make([]byte, DefaultWinSize/8),
		log:              qc.log,
	}

	go r.runLoop()
	r.log.Info("QVIC NewReceiverMgr started!")
	return r
}

func (rv *ReceiverMgr) runLoop() {
	timerTick := time.NewTicker(time.Second)
	timerBitmap := time.NewTicker(30 * time.Millisecond)
	rv.loopWG.Add(1)
	defer rv.loopWG.Done()
needQuit:
	for {
		select {
		case <-timerTick.C:
			rv.lock.Lock()
			if time.Now().Sub(rv.lastHBTime) > maxHearbeatInterval {
				// heartbeat msg is not received for a long time
				if atomic.CompareAndSwapInt32(&rv.qconn.status, StatusConnected, StatusClosing) {
					select {
					case <-rv.qconn.quit:
					default:
						close(rv.qconn.quit)
					}
				}
				rv.lock.Unlock()
				break needQuit
			}
			rv.lock.Unlock()
		case <-timerBitmap.C:
			rv.sendBitmap()
		case <-rv.qconn.quit:
			break needQuit
		}
	}
	rv.log.Debug("ReceiverMgr runLoop OUT")
}

// composeBitmapMsg composes bitmap message head according to input parameters, and returns constant head length
func (rv *ReceiverMgr) composeBitmapMsg(data []byte, curTick, startSeq, toSeq uint32) int {
	binary.BigEndian.PutUint32(data[0:4], rv.qconn.magic)
	data[4] = byte(PackTypeChannel << 4)
	data[5] = msgChannelBitmap
	binary.BigEndian.PutUint32(data[6:10], rv.curSeq)
	binary.BigEndian.PutUint32(data[10:14], rv.maxSeq)
	binary.BigEndian.PutUint16(data[14:16], rv.least_peer_lastSeqSendTick)
	binary.BigEndian.PutUint16(data[16:18], uint16(curTick-rv.least_local_RecvedTick))
	binary.BigEndian.PutUint32(data[18:22], startSeq)
	binary.BigEndian.PutUint32(data[22:26], toSeq)
	binary.BigEndian.PutUint32(data[26:30], rv.startSeq)
	return 30
}

// sendBitmap sends bitmap message to remote peer.
func (rv *ReceiverMgr) sendBitmap() {
	if !rv.bHasPacketRecved {
		return
	}

	rv.lock.Lock()
	defer rv.lock.Unlock()
	curTick := uint32(time.Now().UnixNano() / (1000 * 1000))
	toSeq := rv.maxSeq - ((rv.maxSeq + 1) % uint32(rv.bundleNum))
	diff := curTick - rv.maxSeqNoLocalTick
	maxTick := rv.maxSeqNoCreateTick
	if diff > uint32(rv.qconn.jitter) {
		toSeq = rv.maxSeq
	}

	data := make([]byte, 1500)
	if !rv.qconn.isEqAndBig_32(toSeq, rv.curSeq) {
		startSeq := (rv.curSeq >> 3) << 3
		len := rv.composeBitmapMsg(data[0:], curTick, startSeq, startSeq)
		rv.udpfd.WriteTo(data[0:len], rv.qconn.RemoteAddr())
		return
	}

	if diff < uint32(rv.qconn.jitter) {
		diff = uint32(rv.qconn.jitter) - diff
		cnt := uint32(0)
		for ; toSeq != rv.curSeq; toSeq-- {
			item := &rv.packItems[toSeq%rv.winSize]
			cnt++
			if item.p == nil {
				continue
			}
			if item.p.isRecovered == 1 {
				continue
			}
			if (maxTick - item.p.createTick) > uint16(diff) {
				break
			}
		}
		if rv.qconn.isEqAndBig_32(rv.curSeq, toSeq) {
			startSeq := (rv.curSeq >> 3) << 3
			len := rv.composeBitmapMsg(data[0:], curTick, startSeq, startSeq)
			rv.udpfd.WriteTo(data[0:len], rv.qconn.RemoteAddr())
			return
		}
	}

	startSeq := (rv.curSeq >> 3) << 3
	total := toSeq - startSeq + 1
	if total == 0 {
		len := rv.composeBitmapMsg(data[0:], curTick, startSeq, startSeq)
		rv.udpfd.WriteTo(data[0:len], rv.qconn.RemoteAddr())
		return
	}

	per := uint32(8 * 1024) //max bitmap length for every message.
	cnt := total / per      // send times
	if total%per != 0 {
		cnt++
	}

	preStartSeq := startSeq
	for i := uint32(0); i < cnt; i++ {
		startSeq = preStartSeq + i*per
		num := per
		if (i+1)*per > total {
			num = total - i*per
			if num%8 != 0 {
				num = (num >> 3) << 3
				num += 8
			}
		}
		curToSeq := startSeq + num - 1

		len := rv.composeBitmapMsg(data[0:], curTick, startSeq, curToSeq)
		rv.getBitmap(data[len:], startSeq, num)
		rv.udpfd.WriteTo(data[0:uint32(len)+num/8], rv.qconn.RemoteAddr())
	}
}

func (rv *ReceiverMgr) onHeartBeat(data []byte) {
	rv.lock.Lock()
	defer rv.lock.Unlock()
	rv.lastHBTime = time.Now()
}

func (rv *ReceiverMgr) onRecvVPacket(pack *VPacket, curTick uint32, packType int) {
	rv.lock.Lock()
	defer rv.lock.Unlock()
	item := &rv.packItems[pack.seq%rv.winSize]
	if rv.qconn.isEqAndBig_32(pack.seq, rv.startSeq) && rv.qconn.isEqAndBig_32(rv.startSeq+rv.winSize-1, pack.seq) {
		// pack's seq is in valid range.
		if !rv.bHasPacketRecved {
			rv.bHasPacketRecved = true
		}
		if rv.qconn.isEqAndBig_32(pack.seq, rv.maxSeq) {
			rv.maxSeq, rv.maxSeqNoCreateTick = pack.seq, pack.createTick
			rv.maxSeqNoLocalTick = curTick
		}

		if packType == PackTypeFEC {
			if item.pFECInfo == nil {
				item.pFECInfo = NewFECInfo()
			}
			item.pFECInfo.seq = pack.seq
			value := 1 << (7 - uint(pack.fecIdx))
			item.pFECInfo.fecFlag |= byte(value)
			item.pFECInfo.fecPackets[pack.fecIdx] = pack
			for rv.tryRecoverFec(pack.seq, curTick) {
			}
			for item = &rv.packItems[rv.curSeq%rv.winSize]; item.p != nil; {
				rv.curSeq++
				item = &rv.packItems[rv.curSeq%rv.winSize]
			}
			return
		}

		rv.least_peer_lastSeqSendTick = pack.lastSeqSendTick
		rv.least_local_RecvedTick = curTick
		if item.p != nil {
			return
		}

		item.p = pack
		rv.setBitmap(pack.seq, true)
		for rv.tryRecoverFec(pack.seq, curTick) {
		}

		for item = &rv.packItems[rv.curSeq%rv.winSize]; item.p != nil; {
			rv.curSeq++
			item = &rv.packItems[rv.curSeq%rv.winSize]
		}
	}
}

func (rv *ReceiverMgr) setBitmap(seq uint32, val bool) {
	idx := seq % rv.winSize
	indx, bitIndx := idx>>3, idx&7
	if val {
		rv.bitmap[indx] |= 1 << (7 - bitIndx)
	} else {
		rv.bitmap[indx] &^= 1 << (7 - bitIndx)
	}
}

func (rv *ReceiverMgr) tryRecoverFec(packSeq, curTick uint32) bool {
	beginSeq := packSeq - (packSeq % uint32(rv.bundleNum))
	lastSeq := beginSeq + uint32(rv.bundleNum) - 1
	lastItem := &rv.packItems[lastSeq%rv.winSize]
	fecInfo := lastItem.pFECInfo
	if beginSeq < rv.startSeq || fecInfo == nil || fecInfo.seq != lastSeq {
		return false
	}

	data := make([]byte, rv.bundleNum/8)
	rv.getBitmap(data, beginSeq, uint32(rv.bundleNum))

	bitVec := new(VBitVec)
	bitVec.Attatch(data, uint(rv.bundleNum), fecInfo.fecFlag)
	pC := rv.fecHelper.GetRecoverInfo(bitVec)
	if pC == nil {
		return false
	}

	fecPack := new(VPacket)
	fecPack.data = make([]byte, rv.qconn.packDataSize)
	var recoverSeq uint32

	for i := 0; i < rv.bundleNum; i++ {
		if !pC.GetBit(uint(i)) {
			continue
		}
		if !bitVec.GetBit(uint(i)) {
			recoverSeq = beginSeq + uint32(i)
			continue
		}
		curSeq := beginSeq + uint32(i)
		pack := rv.packItems[curSeq%rv.winSize].p

		for j := 0; j < pack.dataLen; j++ {
			fecPack.data[j] = fecPack.data[j] ^ pack.data[j]
		}
	}

	for i := 0; i < 8; i++ {
		if !pC.GetFlagBit(uint(i)) {
			continue
		}
		pack := fecInfo.fecPackets[i]
		for j := 0; j < pack.dataLen; j++ {
			fecPack.data[j] = fecPack.data[j] ^ pack.data[j]
		}
	}

	fecPack.seq = recoverSeq
	fecPack.packType, fecPack.magic, fecPack.createTick = byte(PackTypeData), rv.qconn.magic, uint16(curTick)
	fecPack.dataLen = int(binary.BigEndian.Uint16(fecPack.data[0:2])) + 2
	fecPack.isRecovered = 1
	fecItem := &rv.packItems[fecPack.seq%rv.winSize]
	fecItem.p = fecPack
	rv.setBitmap(fecPack.seq, true)
	rv.log.Debug("ReceiverMgr tryRecoverFec ok seq=%d", fecPack.seq)
	return true
}

func (rv *ReceiverMgr) getBitmap(data []byte, beginPackNo, packNum uint32) {
	if beginPackNo%8 != 0 || packNum%8 != 0 {
		rv.log.Fatal("qvic getBitmap, parameters not valid")
		return
	}

	endPackNo := beginPackNo + packNum - 1
	startIdx := beginPackNo % rv.winSize
	endIdx := endPackNo % rv.winSize
	indx := startIdx >> 3
	if endIdx > startIdx {
		copy(data[0:], rv.bitmap[indx:indx+packNum/8])
		return
	}

	toEndLen := rv.winSize/8 - indx
	copy(data[0:], rv.bitmap[indx:])
	copy(data[toEndLen:], rv.bitmap[0:packNum/8-toEndLen])
	return
}

// doDel removes packets from packItems and modifies bitmap in the same time.
func (rv *ReceiverMgr) doDel() {
	if rv.qconn.isEqAndBig_32(rv.startSeq+uint32(rv.bundleNum)*2, rv.curSeq) {
		return
	}
	if rv.qconn.isEqAndBig_32(rv.startSeq+uint32(rv.bundleNum)*2, rv.curReadSeq) {
		return
	}

	toDel := rv.curReadSeq - (rv.startSeq + uint32(rv.bundleNum))
	for i := uint32(0); i < toDel; i++ {
		seq := rv.startSeq + i
		item := &rv.packItems[seq%rv.winSize]
		item.fecCreateTick, item.p = 0, nil
		if item.pFECInfo != nil {
			item.pFECInfo.Clear()
		}
		rv.setBitmap(seq, false)
	}
	rv.startSeq += toDel
	rv.log.Debug("qvic recvmgr.doDel toDel:%d curSeq=%d startSeq=%d readSeq=%d maxSeq=%d", toDel, rv.curSeq, rv.startSeq, rv.curReadSeq, rv.maxSeq)
}

func (rv *ReceiverMgr) close() {
	rv.loopWG.Wait()
}

func (rv *ReceiverMgr) tryReadData(data []byte) int {
	rv.lock.Lock()
	defer rv.lock.Unlock()
	curPos, canRead := 0, len(data)

	item := &rv.packItems[rv.curReadSeq%rv.winSize]
	for item.p != nil && canRead > 0 {
		pack := item.p
		curPackLeftLen := pack.dataLen - pack.myPos - 2
		if curPackLeftLen == 0 {
			rv.curReadSeq++
			item = &rv.packItems[rv.curReadSeq%rv.winSize]
			continue
		}
		readLen := curPackLeftLen
		if readLen > canRead {
			readLen = canRead
		}
		copy(data[curPos:], pack.data[pack.myPos+2:pack.myPos+2+readLen])
		curPos += readLen
		pack.myPos += readLen
		canRead -= readLen
		if readLen == curPackLeftLen {
			rv.curReadSeq++
			item = &rv.packItems[rv.curReadSeq%rv.winSize]
			continue
		}
	}
	rv.doDel()
	return curPos
}
