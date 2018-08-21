package p2p

import (
	"net"
	"testing"

	"github.com/magiconair/properties/assert"
)

func newConnection() (*connection, net.Listener, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, err
	}

	c, err := net.Dial(ln.Addr().Network(), ln.Addr().String())
	if err != nil {
		return nil, nil, err
	}
	return &connection{fd: c}, ln, nil
}

func Test_connection(t *testing.T) {
	con, ln, err := newConnection()
	defer ln.Close()
	defer con.close()
	assert.Equal(t, err, nil)

	randStr1 := getRandomString(zipBytesLimit * 10)
	msg1 := newMessage(randStr1)
	msg1Copy := *msg1

	err = con.WriteMsg(&msg1Copy)
	assert.Equal(t, err, nil)

	fd1, err := ln.Accept()
	assert.Equal(t, err, nil)

	con1 := connection{fd: fd1}
	msg2, err := con1.ReadMsg()
	assert.Equal(t, err, nil)
	assert.Equal(t, msg2.Payload, msg1.Payload)
	assert.Equal(t, string(msg2.Payload), randStr1)

	randStr2 := getRandomString(10)
	msg1 = newMessage(randStr2)

	err = con.WriteMsg(msg1)
	assert.Equal(t, err, nil)

	msg3, err := con1.ReadMsg()
	assert.Equal(t, err, nil)
	assert.Equal(t, msg3.Payload, msg1.Payload)
	result := string(msg3.Payload)
	assert.Equal(t, result == randStr2, true)
}
