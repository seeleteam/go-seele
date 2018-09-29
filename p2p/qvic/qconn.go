/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package qvic

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/seeleteam/go-seele/log"
)

// Qvic const
const (
	StatusNone       = 0
	StatusConnecting = 1
	StatusConnected  = 2
	StatusClosing    = 3
	StatusClosed     = 4

	msgHandshake        byte = 1
	msgHandshakeAck     byte = 2
	msgHandshakeErr     byte = 3
	msgQConnClose       byte = 4
	msgChannelHeartBeat byte = 1
	msgChannelBitmap    byte = 2

	DefaultFECBundle = 16 // default packet number every bundle.
	DefaultFECLen    = 3  // default extra packets number every bundle.
	DefaultJitter    = 40
	VPacketDataLen   = 1380

	DefaultWinSize          = 4096 // default window size for QConn.
	DefaultRecvPacketLength = 2048

	ExtraLenPaddingInData = 2
)

// QConn represents a qvic connection, implements net.Conn interface.
type QConn struct {
	lock                sync.Mutex // protects running
	mgr                 *QvicMgr
	myFD, peerFD        uint16
	localAddr, peerAddr net.Addr
	udpFD               *net.UDPConn // UDPConn used by QConn
	quit                chan struct{}

	fecHelper                  *FECHelper
	sendStartSeq, peerStartSeq uint32
	status                     int32
	magic                      uint32
	packDataSize               int
	wg                         sync.WaitGroup
	log                        *log.SeeleLog

	senderMgr     *SenderMgr
	receiverMgr   *ReceiverMgr
	jitter        uint16
	timeLock      sync.Mutex
	readDeadLine  time.Time
	writeDeadLine time.Time
}

// NewQConn creates QvicMgr object.
func NewQConn(mgr *QvicMgr) (*QConn, error) {
	rand.Seed(time.Now().UnixNano())
	q := &QConn{
		quit:         make(chan struct{}),
		status:       StatusNone,
		mgr:          mgr,
		fecHelper:    new(FECHelper),
		packDataSize: VPacketDataLen,
		log:          mgr.log,
		sendStartSeq: (rand.Uint32() >> 4) << 4,
		jitter:       DefaultJitter,
	}
	q.fecHelper.Init(DefaultFECBundle)
	q.readDeadLine = time.Now()
	q.writeDeadLine = q.readDeadLine

	mgr.lock.Lock()
	defer mgr.lock.Unlock()
	fd := mgr.selectFreeSlot()
	if fd == 0 {
		q.log.Info("NewQConn called but selectFreeSlot returns fd=0")
		return nil, errSlotsNotEnough
	}

	if err := q.qvicBind(); err != nil {
		q.log.Info("NewQConn called but qvicBind error, %s", err)
		return nil, err
	}

	mgr.slots[fd] = q
	q.myFD = fd
	q.log.Info("NewQConn called fd=%d", fd)
	return q, nil
}

func (qc *QConn) recvLoop(notifyCh chan struct{}) {
	defer qc.wg.Done()
	qc.log.Info("qconn recvLoop start")
	bNeedNotify := (notifyCh != nil)

needQuit:
	for {
		data := make([]byte, DefaultRecvPacketLength)
		n, remoteAddr, err := qc.udpFD.ReadFromUDP(data)
		if err != nil {
			select {
			case <-qc.quit:
				break needQuit
			default:
			}
			continue
		}

		b := data[:n]
		qc.handleMsg(remoteAddr, b)

		if bNeedNotify && atomic.LoadInt32(&qc.status) != StatusConnecting {
			notifyCh <- struct{}{}
			bNeedNotify = false
		}
	}

	qc.log.Info("QConn recvLoop out. myFD=%d", qc.myFD)
}

func (qc *QConn) handleMsg(from *net.UDPAddr, data []byte) {
	if len(data) < 5 {
		return
	}
	if magic := binary.BigEndian.Uint32(data[:4]); magic != qc.magic {
		return
	}
	ptType := data[4] >> 4
	if int(ptType) == PackTypeControl {
		msgType := data[5]
		switch msgType {
		case msgHandshakeAck:
			if atomic.LoadInt32(&qc.status) != StatusConnecting {
				break
			}
			qc.peerAddr = from
			qc.peerFD = binary.BigEndian.Uint16(data[6:8])
			qc.peerStartSeq = binary.BigEndian.Uint32(data[8:12])
			qc.senderMgr = NewSenderMgr(qc)
			qc.receiverMgr = NewReceiverMgr(qc)
			atomic.StoreInt32(&qc.status, StatusConnected)
			qc.log.Debug("qconn recved msgHandshakeAck myfd=%d", qc.myFD)

		case msgHandshakeErr, msgQConnClose:
			if atomic.CompareAndSwapInt32(&qc.status, StatusConnected, StatusClosing) {
				select {
				case <-qc.quit:
				default:
					close(qc.quit)
				}
			}
			qc.log.Debug("qconn recved msgHandshakeErr or msgQConnClose. msg=%d myfd=%d", msgType, qc.myFD)
		}
	}

	// do not handle below messages if status not match
	if atomic.LoadInt32(&qc.status) != StatusConnected {
		return
	}
	curTick := time.Now().UnixNano() / (1000 * 1000)
	if int(ptType) == PackTypeChannel {
		msgType := data[5]
		switch msgType {
		case msgChannelHeartBeat:
			qc.receiverMgr.onHeartBeat(data)
		case msgChannelBitmap:
			qc.senderMgr.onBitmap(data, uint32(curTick))
		}
	}

	if int(ptType) == PackTypeData || int(ptType) == PackTypeFEC {
		if len(data) < VPacketHeadLen {
			// not a valid packet
			return
		}
		pack := new(VPacket)
		pack.ParseData(data[0:])
		qc.receiverMgr.onRecvVPacket(pack, uint32(curTick), int(ptType))
	}
}

func (qc *QConn) dialTimeout(addr string, timeout time.Duration) error {
	qc.wg.Add(1)
	defer qc.wg.Done()
	peerMgrAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	atomic.StoreInt32(&qc.status, StatusConnecting)
	qc.magic = rand.Uint32()
	qc.log.Debug("qconn dialTimeout magic=%d", qc.magic)
	notifyCh := make(chan struct{})
	qc.wg.Add(1)
	go qc.recvLoop(notifyCh)

	qc.sendHandShake(peerMgrAddr)
	timer := time.NewTimer(timeout)
	timerSend := time.NewTimer(3 * time.Second) // send handshake message every three seconds

needQuit:
	select {
	case <-timerSend.C:
		qc.sendHandShake(peerMgrAddr)
	case <-notifyCh:
		break needQuit
	case <-timer.C:
		break needQuit
	case <-qc.quit:
		break needQuit
	}

	close(notifyCh)
	if atomic.LoadInt32(&qc.status) != StatusConnected {
		qc.log.Info("qconn dialTimeout ERR. myfd=%d err=%s", qc.myFD, errDailFailed)
		return errDailFailed
	}
	qc.log.Info("qconn dialTimeout OK. myfd=%d", qc.myFD)
	return nil
}

func (qc *QConn) sendHandShake(addr *net.UDPAddr) {
	var data [12]byte
	b := data[0:]
	binary.BigEndian.PutUint32(b[:4], qc.magic)
	b[4] = (byte(PackTypeControl) << 4)
	b[5] = msgHandshake
	binary.BigEndian.PutUint16(b[6:8], qc.myFD)
	binary.BigEndian.PutUint32(b[8:12], qc.sendStartSeq)

	// send three packets repeatly
	for i := 0; i < 3; i++ {
		qc.udpFD.WriteToUDP(data[0:], addr)
	}
	qc.log.Info("qconn sendHandShake 3 times. myfd=%d", qc.myFD)
}

func (qc *QConn) sendHandshakeAck() {
	var data [12]byte
	b := data[0:]
	binary.BigEndian.PutUint32(b[:4], qc.magic)
	b[4] = (byte(PackTypeControl) << 4)
	b[5] = msgHandshakeAck
	binary.BigEndian.PutUint16(b[6:8], qc.myFD)
	binary.BigEndian.PutUint32(b[8:12], qc.sendStartSeq)

	qc.udpFD.WriteToUDP(data[0:], qc.peerAddr.(*net.UDPAddr))
	qc.log.Info("qconn sendHandshakeAck. myfd=%d peerfd=%d", qc.myFD, qc.peerFD)
}

func (qc *QConn) sendMsgClose() {
	var data [6]byte
	b := data[0:]
	binary.BigEndian.PutUint32(b[:4], qc.magic)
	b[4] = (byte(PackTypeControl) << 4)
	b[5] = msgQConnClose
	// send three packets repeatly
	for i := 0; i < 3; i++ {
		qc.udpFD.WriteToUDP(data[0:], qc.peerAddr.(*net.UDPAddr))
	}
	qc.log.Info("qconn sendMsgClose. myfd=%d peerfd=%d", qc.myFD, qc.peerFD)
}

func (qc *QConn) acceptQConn(magic uint32, from *net.UDPAddr, data []byte) {
	qc.lock.Lock()
	defer qc.lock.Unlock()
	if atomic.LoadInt32(&qc.status) != StatusNone {
		return
	}

	qc.magic, qc.peerAddr = magic, from
	qc.peerFD = binary.BigEndian.Uint16(data[6:8])
	qc.peerStartSeq = binary.BigEndian.Uint32(data[8:12])
	qc.sendHandshakeAck()
	qc.receiverMgr = NewReceiverMgr(qc)
	qc.senderMgr = NewSenderMgr(qc)
	atomic.StoreInt32(&qc.status, StatusConnected)
	qc.wg.Add(1)
	go qc.recvLoop(nil)
	qc.log.Info("qconn acceptQConn. myfd=%d peerfd=%d magic=%d", qc.myFD, qc.peerFD, qc.magic)
}

func (qc *QConn) qvicBind() error {
	for port := qc.mgr.portStart; port <= qc.mgr.portEnd; port++ {
		addr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
		udpfd, err := net.ListenUDP("udp", addr)
		if err == nil {
			qc.udpFD = udpfd
			qc.localAddr = addr
			return nil
		}
	}

	return errPortNotEnough
}

func (qc *QConn) Read(b []byte) (readLen int, err error) {
	if len(b) == 0 {
		return 0, nil
	}

	if atomic.LoadInt32(&qc.status) != StatusConnected {
		return 0, errQConnInvalid
	}

	if readLen = qc.receiverMgr.tryReadData(b); readLen > 0 {
		return readLen, nil
	}

	tNow := time.Now()
	qc.timeLock.Lock()
	needWait := qc.readDeadLine.Sub(tNow)
	qc.timeLock.Unlock()
	var deadLineTicker *time.Timer
	if needWait.Nanoseconds() > 0 {
		deadLineTicker = time.NewTimer(needWait)
		qc.log.Debug("qconn read waittime=needWait %f", needWait.Seconds())
	} else {
		// set a long enough time, if no readDeadLine is set
		deadLineTicker = time.NewTimer(24 * time.Hour)
		qc.log.Debug("qconn read waittime=24h")
	}
	timer := time.NewTicker(200 * time.Microsecond)

needQuit:
	for {
		select {
		case <-timer.C:
			if readLen = qc.receiverMgr.tryReadData(b); readLen > 0 {
				break needQuit
			}
		case <-deadLineTicker.C:
			err = &net.OpError{Op: "read", Net: "qvic", Source: qc.localAddr, Addr: qc.peerAddr, Err: NewQTimeoutError("qconn read timeout")}
			qc.log.Debug("read deadline reached. %s", err)
			break needQuit
		case <-qc.quit:
			if readLen = qc.receiverMgr.tryReadData(b); readLen > 0 {
				return readLen, nil
			}
			err = errQConnInvalid
			qc.log.Debug("qconn read quit %s", err)
			break needQuit
		}
	}

	return readLen, err
}

func (qc *QConn) Write(b []byte) (n int, err error) {
	if atomic.LoadInt32(&qc.status) != StatusConnected {
		return 0, errQConnInvalid
	}

	tNow := time.Now()
	qc.timeLock.Lock()
	needWait := qc.writeDeadLine.Sub(tNow)
	qc.timeLock.Unlock()
	var deadLineTicker *time.Timer
	if needWait.Nanoseconds() > 0 {
		deadLineTicker = time.NewTimer(needWait)
	} else {
		// set a long enough time, if no wirteDeadLine is set
		deadLineTicker = time.NewTimer(24 * time.Hour)
	}
	timer := time.NewTicker(200 * time.Microsecond)
	curPos, needSend := 0, len(b)
	var errRet error
needQuit:
	for {
		select {
		case <-timer.C:
			curTick := time.Now().UnixNano() / (1000 * 1000)
			for needSend > 0 {
				pack := new(VPacket)
				pack.packType, pack.magic, pack.createTick = byte(PackTypeData), qc.magic, uint16(curTick)
				roundLen := qc.packDataSize - ExtraLenPaddingInData
				if roundLen > needSend {
					roundLen = needSend
				}

				pack.dataLen = roundLen + ExtraLenPaddingInData
				pack.data = make([]byte, roundLen+ExtraLenPaddingInData)
				binary.BigEndian.PutUint16(pack.data[0:ExtraLenPaddingInData], uint16(roundLen))
				copy(pack.data[ExtraLenPaddingInData:], b[curPos:curPos+roundLen])
				if !qc.senderMgr.trySendPacket(pack, uint32(curTick)) {
					break
				}
				curPos, needSend = curPos+roundLen, needSend-roundLen
			}
			if needSend == 0 {
				break needQuit
			}
		case <-deadLineTicker.C:
			errRet = &net.OpError{Op: "write", Net: "qvic", Source: qc.localAddr, Addr: qc.peerAddr, Err: NewQTimeoutError("qconn write timeout")}
			break needQuit
		case <-qc.quit:
			errRet = errQConnInvalid
			break needQuit
		}
	}

	if curPos > 0 {
		qc.senderMgr.tryRelaySendPackets()
	}
	return curPos, errRet
}

// Close the qvic connection
func (qc *QConn) Close() error {
	if atomic.LoadInt32(&qc.status) == StatusConnected {
		qc.sendMsgClose()
	}
	atomic.StoreInt32(&qc.status, StatusClosing)

	select {
	case <-qc.quit:
	default:
		close(qc.quit)
	}
	if qc.senderMgr != nil {
		qc.senderMgr.close()
	}
	if qc.receiverMgr != nil {
		qc.receiverMgr.close()
	}

	if qc.udpFD != nil {
		qc.udpFD.Close()
	}
	qc.wg.Wait()

	qc.mgr.lock.Lock()
	defer qc.mgr.lock.Unlock()
	delete(qc.mgr.magicMap, qc.magic)
	qc.mgr.slots[qc.myFD] = nil
	atomic.StoreInt32(&qc.status, StatusClosed)
	qc.log.Info("qconn closed. myfd=%d", qc.myFD)
	return nil
}

// LocalAddr returns the local network address.
func (qc *QConn) LocalAddr() net.Addr {
	return qc.localAddr
}

// RemoteAddr returns the remote network address.
func (qc *QConn) RemoteAddr() net.Addr {
	return qc.peerAddr
}

func (qc *QConn) SetDeadline(t time.Time) error {
	qc.timeLock.Lock()
	defer qc.timeLock.Unlock()
	qc.readDeadLine, qc.writeDeadLine = t, t
	return nil
}

func (qc *QConn) SetReadDeadline(t time.Time) error {
	qc.timeLock.Lock()
	defer qc.timeLock.Unlock()
	qc.readDeadLine = t
	return nil
}

func (qc *QConn) SetWriteDeadline(t time.Time) error {
	qc.timeLock.Lock()
	defer qc.timeLock.Unlock()
	qc.writeDeadLine = t
	return nil
}

func (qc *QConn) isEqAndBig_32(first, second uint32) bool {
	var maxDiff uint32 = 512 * 1024
	if first > second {
		return (second - first) >= maxDiff
	}
	return (first - second) <= maxDiff
}

func (qc *QConn) isEqAndBig_16(first, second uint16) bool {
	if (uint16)(first-second) < 32767 {
		return true
	}
	return false
}
