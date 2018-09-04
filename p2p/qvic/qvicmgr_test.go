/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package qvic

import (
	"fmt"
	"net"
	"reflect"
	"testing"
	"time"
)

func CreateServer(tcpAddress string, qvicAddress string) (qmgr *QvicMgr, err error) {
	qmgr = NewQvicMgr()
	err = qmgr.Listen(tcpAddress, qvicAddress)
	return
}

//Test_qvicmgr tests QvicMgr's core functions
func Test_qvicmgr(t *testing.T) {
	qmgr, err := CreateServer("127.0.0.1:8001", "127.0.0.1:8001")
	if qmgr == nil {
		fmt.Println("CreateServer error", err)
		t.Fail()
	}

	connCnt, errCnt := net.DialTimeout("tcp", "127.0.0.1:8001", time.Second)
	if errCnt != nil {
		fmt.Println("connect to local server error ", errCnt)
		t.Fail()
	}

	connSvr, errSvr := qmgr.Accept()
	if errSvr != nil {
		fmt.Println("connect to local server error ", errSvr)
		t.Fail()
	}

	if reflect.TypeOf(connSvr).String() != "*net.TCPConn" {
		t.Fail()
	}

	connCnt.Close()
	connSvr.Close()
	qmgr.Close()
}

func Test_qvicmgr_qconn(t *testing.T) {
	qmgrSvr, _ := CreateServer("", "127.0.0.1:8001")
	qmgrCnt, _ := CreateServer("", "127.0.0.1:8002")

	// dails to a invalid port
	connCnt, _ := qmgrCnt.DialTimeout("qvic", "127.0.0.1:9001", 2*time.Second)
	fmt.Println(connCnt)
	if connCnt != nil {
		t.Fail()
	}

	// dails to a valid port
	connCnt, _ = qmgrCnt.DialTimeout("qvic", "127.0.0.1:8001", 2*time.Second)
	connSvr, _ := qmgrSvr.Accept()

	if reflect.TypeOf(connCnt).String() != "*qvic.QConn" {
		t.Fail()
	}
	if reflect.TypeOf(connSvr).String() != "*qvic.QConn" {
		t.Fail()
	}

	qmgrCnt.Close()
	qmgrSvr.Close()
}
