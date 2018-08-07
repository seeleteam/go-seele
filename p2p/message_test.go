package p2p

import (
	"math/rand"
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
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

func Test_message(t *testing.T) {
	pl1 := getRandomString(zipBytesLimit - 50)
	m1 := newMessage(pl1)

	err := m1.Zip()
	assert.Equal(t, err, nil)
	assert.Equal(t, string(m1.Payload[:len(m1.Payload)-1]), pl1)

	err = m1.UnZip()
	assert.Equal(t, err, nil)
	assert.Equal(t, string(m1.Payload), pl1)

	pl2 := getRandomString(zipBytesLimit + 50)
	m2 := newMessage(pl2)

	err = m2.Zip()
	assert.Equal(t, err, nil)
	assert.Equal(t, len(m2.Payload[:len(m2.Payload)-1]) < len([]byte(pl2)), true)

	err = m2.UnZip()
	assert.Equal(t, err, nil)
	assert.Equal(t, string(m2.Payload), pl2)
}
