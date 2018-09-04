package p2p

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

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
	err = con1.readFullo(readBuff, readTimeout)
	assert.Equal(t, err, nil)
	assert.Equal(t, readBuff, writeBuff)

	// Case 2: read with empty buff
	readBuff1 := make([]byte, 0)
	err = con1.readFullo(readBuff1, readTimeout)
	assert.Equal(t, err, nil)
	assert.Equal(t, len(readBuff1), 0)

	// Case 3: write 10 bytes and read them with 20 bytes buff
	err = con.writeFull(writeBuff)
	assert.Equal(t, err, nil)

	readBuff2 := make([]byte, 20)
	err = con1.readFullo(readBuff2, readTimeout)
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
	err = con1.readFullo(readBuff3, readTimeout)
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

	con1 := connection{fd: fd1}
	randStr1 := getRandomString(zipBytesLimit * 10)
	msg1 := newMessage(randStr1)
	msg1Copy := *msg1
	var nounceCnt uint64
	binary.Read(rand.Reader, binary.BigEndian, &nounceCnt)
	con.magic = nounceCnt
	// case 1: client write with nounceCnt
	err = con.WriteMsg(&msg1Copy)
	assert.Equal(t, err, nil)

	msg2, err := con1.ReadMsg()
	assert.Equal(t, err, nil)
	assert.Equal(t, msg2.Payload, msg1.Payload)
	assert.Equal(t, string(msg2.Payload), randStr1)
	fmt.Println("case 1:", err)
	// case 2: server write with nounceCnt
	randStr2 := getRandomString(10)
	msg1 = newMessage(randStr2)

	con1.magic = nounceCnt
	err = con1.WriteMsg(msg1)
	assert.Equal(t, err, nil)

	con.checkMagic = true
	msg3, err := con.ReadMsg()
	assert.Equal(t, err, nil)
	assert.Equal(t, msg3.Payload, msg1.Payload)
	result := string(msg3.Payload)
	assert.Equal(t, result == randStr2, true)
	fmt.Println("case 2:", err)
	// case 3: cleint server unconsistent
	randStr1 = getRandomString(zipBytesLimit * 10)
	msg1 = newMessage(randStr1)
	msg1Copy = *msg1
	binary.Read(rand.Reader, binary.BigEndian, &nounceCnt)
	con.magic = nounceCnt

	err = con.WriteMsg(&msg1Copy)
	assert.Equal(t, err, nil)

	msg2, err = con1.ReadMsg()
	assert.Equal(t, err, nil)
	assert.Equal(t, msg2.Payload, msg1.Payload)
	assert.Equal(t, string(msg2.Payload), randStr1)

	randStr2 = getRandomString(10)
	msg1 = newMessage(randStr2)

	con1.magic = nounceCnt + 1
	err = con1.WriteMsg(msg1)
	assert.Equal(t, err, nil)

	con.checkMagic = true
	msg3, err = con.ReadMsg()
	assert.Equal(t, err, errors.New("Failed to wait magic"))
	assert.Equal(t, msg3, &Message{})

	// case 4: too big size greater than 8M bytes
	randStr1 = getRandomString(zipBytesLimit * 1024)
	msg1 = newMessage(randStr1)
	msg1Copy = *msg1
	binary.Read(rand.Reader, binary.BigEndian, &nounceCnt)
	con.magic = nounceCnt

	err = con.WriteMsg(&msg1Copy)
	assert.Equal(t, err, nil)

	msg2, err = con1.ReadMsg()
	assert.Equal(t, err, errors.New("Failed to get data, size is too big"))
	assert.Equal(t, msg2, &Message{})
}
