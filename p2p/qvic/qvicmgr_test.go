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

func CreateServer() (qmgr *QvicMgr, err error) {
	qmgr = NewQvicMgr()
	err = qmgr.Listen("127.0.0.1:8001", "127.0.0.1:8001")
	return
}

//Test_fechelper_mem create random string in memory; simulate packet loss, and try recover by fec
func Test_qvicmgr(t *testing.T) {
	qmgr, err := CreateServer()
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
