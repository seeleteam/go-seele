/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package qvic

import (
	"net"
	"sync"
	"time"
)

// QConn represents a qvic connection, implements net.Conn interface.
type QConn struct {
	lock sync.Mutex // protects running
	quit chan struct{}
}

func (qc *QConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (qc *QConn) Write(b []byte) (n int, err error) {
	return 0, nil
}

func (qc *QConn) Close() error {
	return nil
}

// LocalAddr returns the local network address.
func (qc *QConn) LocalAddr() net.Addr {
	return nil
}

// RemoteAddr returns the remote network address.
func (qc *QConn) RemoteAddr() net.Addr {
	return nil
}

func (qc *QConn) SetDeadline(t time.Time) error {
	return nil
}

func (qc *QConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (qc *QConn) SetWriteDeadline(t time.Time) error {
	return nil
}
