package p2p

import (
	"crypto/rand"
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/seeleteam/go-seele/log"
	"github.com/stretchr/testify/assert"
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
	return &connection{fd: c, log: log.GetLogger("p2p")}, ln, nil
}

func Test_Conn_ReadFullAndWriteFull(t *testing.T) {
	readTimeout := 1 * time.Second

	con, ln, err := newConnection()
	defer ln.Close()
	defer con.close()
	assert.Equal(t, err, nil)

	fd1, err := ln.Accept()
	assert.Equal(t, err, nil)
	con1 := connection{fd: fd1}

	// Case 1: write 10 bytes and read them
	writeBuff := []byte(getRandomString(10))
	err = con.writeFull(writeBuff)
	assert.Equal(t, err, nil)

	readBuff := make([]byte, 10)
	err = con1.readFullTimeout(readBuff, readTimeout)
	assert.Equal(t, err, nil)
	assert.Equal(t, readBuff, writeBuff)

	// Case 2: read with empty buff
	readBuff1 := make([]byte, 0)
	err = con1.readFullTimeout(readBuff1, readTimeout)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(readBuff1), 0)

	// Case 3: write 10 bytes and read them with 20 bytes buff
	err = con.writeFull(writeBuff)
	assert.Equal(t, err, nil)

	readBuff2 := make([]byte, 20)
	err = con1.readFullTimeout(readBuff2, readTimeout)
	netErr, _ := err.(net.Error)
	assert.Equal(t, netErr.Timeout(), true)

	assert.Equal(t, readBuff2[0:10], writeBuff)
	emptyBuff := make([]byte, 10)
	assert.Equal(t, readBuff2[10:], emptyBuff[:])

	// Case 4: write 20 bytes and read them with 10 bytes buff
	writeBuff = []byte(getRandomString(20))
	err = con.writeFull(writeBuff)
	assert.Equal(t, err, nil)

	readBuff3 := make([]byte, 10)
	err = con1.readFullTimeout(readBuff3, readTimeout)
	assert.Equal(t, err, nil)
	assert.Equal(t, readBuff3[0:], writeBuff[0:10])
}

func Test_connection(t *testing.T) {
	con, ln, err := newConnection()
	defer ln.Close()
	defer con.close()
	assert.Equal(t, err, nil)

	fd1, err := ln.Accept()
	assert.Equal(t, err, nil)

	con1 := connection{fd: fd1, log: log.GetLogger("p2p")}
	randStr1 := getRandomString(zipBytesLimit * 10)
	msg1 := newMessage(randStr1)
	msg1Copy := *msg1
	var nounceCnt uint64
	binary.Read(rand.Reader, binary.BigEndian, &nounceCnt)

	// case 1: client consitent with server
	err = con.WriteMsg(&msg1Copy)
	assert.Equal(t, err, nil)

	msg2, err := con1.ReadMsg()
	assert.Equal(t, err, nil)
	assert.Equal(t, msg2.Payload, msg1.Payload)
	assert.Equal(t, string(msg2.Payload), randStr1)

	// case 2: server write with magic
	randStr2 := getRandomString(10)
	msg1 = newMessage(randStr2)
	err = con1.WriteMsg(msg1)
	assert.Equal(t, err, nil)

	// change the magic
	magic = [2]byte{'1', '1'}
	magicNumber = binary.BigEndian.Uint16(magic[:])
	msg3, err := con.ReadMsg()
	assert.Equal(t, err, errMagic)
	assert.Equal(t, msg3, &Message{})

	// case 3: too big size greater than 8M bytes
	randStr1 = getRandomString(zipBytesLimit)
	maxSize = 10
	msg1 = newMessage(randStr1)
	msg1Copy = *msg1
	binary.Read(rand.Reader, binary.BigEndian, &nounceCnt)
	magic = [2]byte{'^', '~'}
	magicNumber = binary.BigEndian.Uint16(magic[:])
	err = con.WriteMsg(&msg1Copy)
	assert.Equal(t, err, nil)

	msg2, err = con1.ReadMsg()
	assert.Equal(t, err, errSize)
	assert.Equal(t, msg2, &Message{})
}
