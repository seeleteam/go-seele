/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package qvic

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/log"
)

var (
	errAlreadyInited   = errors.New("QvicMgr already initialized")
	errInvalidProtocol = errors.New("Invalid connection prototol, must be tcp or qvic")
	errQVICMgrFinished = errors.New("QVIC module has finished")
	errUnknownError    = errors.New("Unknown error")
)

// QvicMgr manages qvic module.
type QvicMgr struct {
	lock         sync.Mutex
	quit         chan struct{}
	acceptChan   chan *acceptInfo
	udpfd        *net.UDPConn
	tcpListenner net.Listener

	loopWG sync.WaitGroup
	log    *log.SeeleLog
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
		acceptChan: make(chan *acceptInfo),
		log:        log.GetLogger("qvic", common.LogConfig.PrintLog),
	}
	q.log.Info("QVIC module started!")
	return q
}

// DialTimeout connects to the address on the named network with a timeout config.
// network parameters must be "tcp" or "qvic" for tcp connection and qvic connection respectively.
func (mgr *QvicMgr) DialTimeout(network, addr string, timeout time.Duration) (net.Conn, error) {
	if network == "tcp" {
		conn, err := net.DialTimeout("tcp", addr, timeout)
		if err != nil {
			mgr.log.Error("connect to a new node err: %s", err)
			if conn != nil {
				conn.Close()
			}
			return nil, err
		}
		mgr.log.Debug("qvic DialTimeout OK! network=%s addr=%s", network, addr)
		return conn, nil
	}

	if network == "qvic" {
		//TODO qvic dial support
	}

	return nil, errInvalidProtocol
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
		go mgr.qvicRun()
	}

	return nil
}

func (mgr *QvicMgr) qvicRun() {
	mgr.loopWG.Add(1)
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

		data = data[:n]
		mgr.handleMsg(remoteAddr, data)
	}
	mgr.log.Info("qvic qvicRun out")
}

func (mgr *QvicMgr) handleMsg(from *net.UDPAddr, data []byte) {
	//TODO handle udp message

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

	// TODO close all qvic connections
}
