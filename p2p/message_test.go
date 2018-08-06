package p2p

import (
	"math/rand"
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
)

func newMessage(payLoad string) *Message {
	return &Message{
		Code:    3,
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
	pl1 := getRandomString(zipLimit - 50)
	m1 := newMessage(pl1)

	err := m1.ZipMessage()
	assert.Equal(t, err, nil)
	assert.Equal(t, m1.ZipCode, uint16(0))
	assert.Equal(t, string(m1.Payload), pl1)

	err = m1.UZipMessage()
	assert.Equal(t, err, nil)
	assert.Equal(t, string(m1.Payload), pl1)

	pl2 := getRandomString(zipLimit + 50)
	m2 := newMessage(pl2)

	err = m2.ZipMessage()
	assert.Equal(t, err, nil)
	assert.Equal(t, m2.ZipCode, ctlMsgZipCode)
	assert.Equal(t, len(m2.Payload) < len([]byte(pl2)), true)

	err = m2.UZipMessage()
	assert.Equal(t, err, nil)
	assert.Equal(t, string(m2.Payload), pl2)
}
