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

const getXTopic = "0x672e793f48f65acb771442258a567e553d1620c0684e1cbd9fe06ee380d1b642"
const getYTopic = "0x1086821eef716a909c39f2efe1e810bcd29246a6da19d04f9fc3f8d2889392e5"

func Test_NewContractEventABI(t *testing.T) {
	currentProjectPath, err := os.Getwd()
	assert.NoError(t, err)
	configFilePath1 := filepath.Join(currentProjectPath, path1)

	// empty abi path
	_, err = NewContractEventABI("")
	assert.Equal(t, err, ErrInvalidArguments)

	// empty events
	_, err = NewContractEventABI(configFilePath1)
	assert.Equal(t, err, ErrInvalidArguments)

	// valid arguments
	c, err := NewContractEventABI(configFilePath1, getX, getY)
	assert.NoError(t, err)
	topicEventNames := map[string]string{
		getXTopic: getX,
		getYTopic: getY,
	}
	assert.Equal(t, c.topicEventNames, topicEventNames)
}
