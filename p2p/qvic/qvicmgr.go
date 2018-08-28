/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package qvic

import (
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/log"
)

const (
	SlotsNum            = 16384 // maximum number of qconn that QvicMgr can hold.
	DefaultPortStart    = 50000
	DefaultPortEnd      = 51023
	maxHearbeatInterval = 12 * time.Second // max heartbeat timeout interval
)

var (
	errAlreadyInited   = errors.New("QvicMgr already initialized")
	errInvalidProtocol = errors.New("Invalid connection prototol, must be tcp or qvic")
	errQVICMgrFinished = errors.New("QVIC module has finished")
	errUnknownError    = errors.New("Unknown error")
	errSlotsNotEnough  = errors.New("QvicMgr's Slots not enough")
	errPortNotEnough   = errors.New("QvicMgr's port range is too small")
	errDailFailed      = errors.New("QConn connects err")
	errQConnInvalid    = errors.New("QConn is not valid")
)

// QvicMgr manages qvic module.
type QvicMgr struct {
	lock               sync.Mutex
	quit               chan struct{}
	acceptChan         chan *acceptInfo
	udpfd              *net.UDPConn
	tcpListenner       net.Listener
	magicMap           map[uint32]*QConn
	portStart, portEnd int      // udp port's range used by QConn
	slots              []*QConn // holds all QConns.
	loopWG             sync.WaitGroup
	log                *log.SeeleLog
}

// acceptInfo represents acceptance information for both tcp and qvic.
type acceptInfo struct {
	conn net.Conn
	err  error
}

// NewQvicMgr creates QvicMgr object.
func NewQvicMgr() *QvicMgr {
	q := &QvicMgr{
		quit:       make(chan struct{}),
		acceptChan: make(chan *acceptInfo, 5),
		slots:      make([]*QConn, SlotsNum),
		portStart:  DefaultPortStart,
		portEnd:    DefaultPortEnd,
		magicMap:   make(map[uint32]*QConn),
		log:        log.GetLogger("qvic"),
	}

	q.log.Info("QVIC module started!")
	return q
}

// DialTimeout connects to the address on the named network with a timeout config.
// network parameters must be "tcp" or "qvic" for tcp connection and qvic connection respectively.
func (mgr *QvicMgr) DialTimeout(network, addr string, timeout time.Duration) (conn net.Conn, err error) {
	if network == "tcp" {
		conn, err = net.DialTimeout("tcp", addr, timeout)
	}

	if network == "qvic" {
		qconn, errQvic := NewQConn(mgr)
		if errQvic != nil {
			return nil, errQvic
		}

		err = qconn.dialTimeout(addr, timeout)
		conn = qconn
	}

	if err != nil {
		conn.Close()
		mgr.log.Debug("qvic failed to connect. addr=%s err=%s", addr, err)
		return nil, err
	}

	return conn, err
}

// selectFreeSlot selects free slot
func (mgr *QvicMgr) selectFreeSlot() uint16 {
	var fd uint16
	for i := 1; i < SlotsNum; i++ {
		if mgr.slots[i] == nil {
			fd = uint16(i)
			break
		}
	}
	return fd
}

type tempError interface {
	Temporary() bool
}

// Listen binds ports and starts listenning for both tcp and qvic.
func (mgr *QvicMgr) Listen(tcpAddress string, qvicAddress string) (err error) {
	if mgr.tcpListenner != nil || mgr.udpfd != nil {
		return errAlreadyInited
	}

	if len(tcpAddress) != 0 {
		mgr.tcpListenner, err = net.Listen("tcp", tcpAddress)
		if err != nil {
			return err
		}
		go func() {
			mgr.loopWG.Add(1)
			defer mgr.loopWG.Done()
			for {
				fd, errConn := mgr.tcpListenner.Accept()
				if tempErr, ok := errConn.(tempError); ok && tempErr.Temporary() {
					continue
				} else if errConn != nil {
					mgr.log.Error("qvic. tcp accept err. %s", errConn)
					break
				}
				mgr.acceptChan <- &acceptInfo{fd, errConn}
			}
			mgr.log.Info("tcp loop accept quit")
		}()
	}

	if len(qvicAddress) != 0 {
		addr, _ := net.ResolveUDPAddr("udp", qvicAddress)
		mgr.udpfd, err = net.ListenUDP("udp", addr)
		if err != nil {
			mgr.log.Error("qvic. qvic-protocol udp listen err. %s", err)
			return err
		}
		mgr.loopWG.Add(1)
		go mgr.qvicRun()
	}

	return nil
}

func (mgr *QvicMgr) qvicRun() {
	defer mgr.loopWG.Done()
	mgr.log.Info("qvic qvicRun start")

needQuit:
	for {
		data := make([]byte, 2048)
		n, remoteAddr, err := mgr.udpfd.ReadFromUDP(data)
		if err != nil {
			mgr.log.Warn("qvicRun read udp failed. %s", err)
			select {
			case <-mgr.quit:
				break needQuit
			default:
			}
			continue
		}

		b := data[:n]
		mgr.handleMsg(remoteAddr, b)
	}
	mgr.log.Info("qvic qvicRun out")
}

func (mgr *QvicMgr) handleMsg(from *net.UDPAddr, data []byte) {
	if len(data) < 5 {
		return
	}

	ptType := data[4] >> 4
	msgType := data[5]
	if int(ptType) != PackTypeControl || msgType != msgHandshake {
		return
	}

	magic := binary.BigEndian.Uint32(data[:4])
	mgr.lock.Lock()
	if qconn, ok := mgr.magicMap[magic]; ok {
		mgr.lock.Unlock()
		mgr.log.Debug("qvic recved inbound qconn request. Already exists, only need sendHandshakeAck")
		qconn.sendHandshakeAck()
		return
	}
	mgr.lock.Unlock()
	// create QConn for inbound qvic connection
	qconn, errQvic := NewQConn(mgr)
	if errQvic != nil {
		mgr.log.Debug("qvic recved inbound qconn request, but call NewQConn err=%s.", errQvic)
		mgr.sendConnectErrMsg(magic, from)
		return
	}

	mgr.lock.Lock()
	mgr.magicMap[magic] = qconn
	mgr.lock.Unlock()

	mgr.log.Debug("qvic recved inbound qconn request, accepts qconn.")
	qconn.acceptQConn(magic, from, data)
	mgr.acceptChan <- &acceptInfo{qconn, nil}
}

func (mgr *QvicMgr) sendConnectErrMsg(magic uint32, from *net.UDPAddr) {
	var data [6]byte
	b := data[0:]
	binary.BigEndian.PutUint32(b[:4], magic)
	b[4] = (byte(PackTypeControl) << 4)
	b[5] = msgHandshakeErr
	mgr.udpfd.WriteToUDP(data[0:], from)
	mgr.log.Debug("qvic sendConnectErrMsg")
}

// Accept gets connection from qvic module if exists.
func (mgr *QvicMgr) Accept() (net.Conn, error) {
	mgr.log.Info("accept start")
	mgr.loopWG.Add(1)
	defer mgr.loopWG.Done()
	select {
	case <-mgr.quit:
		mgr.log.Info("qvic Accept, but received quit message")
		return nil, errQVICMgrFinished
	case acc := <-mgr.acceptChan:
		if acc.err == nil {
			mgr.log.Info("qvic Accepted OK")
		} else {
			mgr.log.Info("qvic Accepted, err=%s", acc.err)
		}
		return acc.conn, acc.err
	}
}

// Close clean for QvicMgr object.
func (mgr *QvicMgr) Close() {
	mgr.log.Info("qvic Close called")
	close(mgr.quit)
	if mgr.tcpListenner != nil {
		mgr.tcpListenner.Close()
	}
	if mgr.udpfd != nil {
		mgr.udpfd.Close()
	}
	mgr.loopWG.Wait()
	close(mgr.acceptChan)

	for _, c := range mgr.magicMap {
		c.Close()
	}
}
