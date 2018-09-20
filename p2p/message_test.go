package p2p

import (
	"compress/gzip"
	"errors"
	"math/rand"
	"testing"
	"time"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core"
	"github.com/stretchr/testify/assert"
)

func newMessage(payLoad string) *Message {
	return &Message{
		Code:    ctlMsgPingCode,
		Payload: []byte(payLoad),
	}
}

func getRandomString(l int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < l; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}

// TestMsgReadWriter is a test implementation of the MsgReadWriter interface.
type TestMsgReadWriter struct {
	reader TestMsgReader
	writer TestMsgWriter
}

type TestMsgReader struct{}
type TestMsgWriter struct{}

func (s TestMsgReader) ReadMsg() (*Message, error) { return nil, nil }
func (s TestMsgWriter) WriteMsg(*Message) error    { return nil }

type TestMsgReadWriterBad struct {
	reader TestMsgReaderBad
	writer TestMsgWriterBad
}

type TestMsgReaderBad struct{}
type TestMsgWriterBad struct{}

func (s TestMsgReaderBad) ReadMsg() (*Message, error) { return nil, nil }
func (s TestMsgWriterBad) WriteMsg(*Message) error    { return errors.New("error") }

func Test_SendMessage(t *testing.T) {
	str := "5aaeb6053f3e94c9b9a09f33669435e7"
	hash := common.StringToHash(str)
	buff := common.SerializePanic(hash)
	var transactionHashMsgCode uint16

	var testMsgReadWriter = TestMsgReadWriter{}
	err := SendMessage(testMsgReadWriter.writer, transactionHashMsgCode, buff)
	assert.Equal(t, err, nil)

	var testMsgReadWriterbad = TestMsgReadWriterBad{}
	err = SendMessage(testMsgReadWriterbad.writer, transactionHashMsgCode, buff)
	assert.Equal(t, err != nil, true)
}

func Test_message(t *testing.T) {
	randStr1 := getRandomString(zipBytesLimit - 50)
	msg1 := newMessage(randStr1)

	err := msg1.Zip()
	assert.Equal(t, err, nil)
	assert.Equal(t, string(msg1.Payload[1:]), randStr1)

	err = msg1.UnZip()
	assert.Equal(t, err, nil)
	assert.Equal(t, string(msg1.Payload), randStr1)

	randStr2 := getRandomString(zipBytesLimit + 50)
	msg2 := newMessage(randStr2)

	err = msg2.Zip()
	assert.Equal(t, err, nil)
	assert.Equal(t, len(msg2.Payload) < len([]byte(randStr2)), true)

	err = msg2.UnZip()
	assert.Equal(t, err, nil)
	assert.Equal(t, string(msg2.Payload), randStr2)

	// Empty payload
	msgEmpty := newMessage("")
	err = msgEmpty.Zip()
	assert.Equal(t, err, nil)
	assert.Equal(t, len(msgEmpty.Payload) == 0, true)

	err = msgEmpty.UnZip()
	assert.Equal(t, err, nil)
	assert.Equal(t, string(msgEmpty.Payload), "")

	// Corrupted data
	randStr := getRandomString(zipBytesLimit + 50)
	msg := newMessage(randStr)
	err = msg.Zip()
	assert.Equal(t, err, nil)
	assert.Equal(t, len(msg.Payload) < len([]byte(randStr)), true)

	msg.Payload[len(msg.Payload)-1] = '\t'
	err = msg.UnZip()
	assert.Equal(t, err, gzip.ErrChecksum)
}

func Benchmark_message_Zip(b *testing.B) {
	randStr := getRandomString(core.BlockByteLimit)

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		msg := newMessage(randStr)

		b.StartTimer()
		msg.Zip()
	}
}

func Benchmark_message_UnZip(b *testing.B) {
	randStr := getRandomString(core.BlockByteLimit)

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		msg := newMessage(randStr)
		msg.Zip()

		b.StartTimer()
		msg.UnZip()
	}
}
