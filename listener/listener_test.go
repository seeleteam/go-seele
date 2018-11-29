/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package listener

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

const path1 = `/testConfig/SimpleEventTest1.abi`
const getX = "getX"
const getY = "getY"

const path2 = `/testConfig/SimpleEventTest2.abi`
const getA = "getA"
const getB = "getB"

const patherr = `SimpleEventTest.abi`

const getATopic = "0xa0acb9dd79e9d920ef642cb67cc5040eb54b29b163936c05777853bc5f4772b0"
const getBTopic = "0xa1c51915e437ec30e58312c6ff1ae0b5e7fc72426b83ddac06c2431e9edc5da1"

func Test_NewListener(t *testing.T) {
	currentProjectPath, err := os.Getwd()
	assert.NoError(t, err)
	configFilePath1 := filepath.Join(currentProjectPath, path1)

	// empty abi path
	_, err = NewListener("")
	assert.Equal(t, err, ErrInvalidArguments)
	// empty events
	_, err = NewListener(configFilePath1)
	assert.Equal(t, err, ErrInvalidArguments)
	// valid arguments
	l, err := NewListener(configFilePath1, getX, getY)
	assert.NoError(t, err)
	assert.Equal(t, l.abiPath, configFilePath1)
	assert.Contains(t, l.eventNames, getX)
	assert.Contains(t, l.eventNames, getY)
}

func Test_Listener_Start(t *testing.T) {
	currentProjectPath, err := os.Getwd()
	assert.NoError(t, err)
	configFilePath1 := filepath.Join(currentProjectPath, path1)
	configFilePath2 := filepath.Join(currentProjectPath, path2)

	// empty event
	l, err := NewListener(configFilePath1, getX, "")
	assert.NoError(t, err)
	assert.Equal(t, l.Start(), ErrNoEvent)
	assert.Equal(t, l.running, int32(0))

	// already running Listener
	l, err = NewListener(configFilePath1, getX)
	assert.NoError(t, err)
	l.running = 1
	assert.Equal(t, l.Start(), ErrListenerIsRunning)

	// error abi path
	l, err = NewListener(patherr, getB)
	assert.NoError(t, err)
	err = l.Start()
	assert.Error(t, err)
	assert.Equal(t, l.running, int32(0))

	// valid arguments
	l, err = NewListener(configFilePath2, getA, getB)
	assert.NoError(t, err)
	assert.NoError(t, l.Start())
	assert.Equal(t, l.running, int32(1))
	var topics []string
	for topic := range l.topics {
		topics = append(topics, topic)
	}
	assert.Contains(t, topics, getATopic)
	assert.Contains(t, topics, getBTopic)
}

func Test_Listener_Stop(t *testing.T) {
	currentProjectPath, err := os.Getwd()
	assert.NoError(t, err)
	configFilePath2 := filepath.Join(currentProjectPath, path2)
	l, err := NewListener(configFilePath2, getA, getB)
	assert.NoError(t, err)
	assert.NoError(t, l.Start())
	assert.Equal(t, l.running, int32(1))
	l.Stop()
	assert.Equal(t, l.running, int32(0))
	assert.Equal(t, len(l.topics), 0)
}
