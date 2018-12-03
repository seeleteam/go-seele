/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package listener

import (
	"encoding/json"
	"math/big"
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
const contract1 = "0x12fe58608430e36ba6bfb0a9bc5623a634530002"

const path2 = `/testConfig/SimpleEventTest2.abi`
const getA = "getA"
const getB = "getB"
const getATopic = "0xa0acb9dd79e9d920ef642cb67cc5040eb54b29b163936c05777853bc5f4772b0"
const getBTopic = "0xa1c51915e437ec30e58312c6ff1ae0b5e7fc72426b83ddac06c2431e9edc5da1"
const contract2 = "0x170677801cb2a9faf387573c7fae61e440480002"

const argString = "abcdefghigklmnopqrstuvwxyzabcdefghigklmnopqrstuvwxyzabcdefghigklmnopqrstuvwxyz"

func Test_NewContractEventABI(t *testing.T) {
	currentProjectPath, err := os.Getwd()
	assert.NoError(t, err)
	configFilePath1 := filepath.Join(currentProjectPath, path1)

	// empty abi path
	_, err = NewContractEventABI("", contract1)
	assert.Equal(t, err, ErrInvalidArguments)

	// empty contract
	_, err = NewContractEventABI(configFilePath1, "", getX, getY)
	assert.Equal(t, err, ErrInvalidArguments)

	// empty events
	_, err = NewContractEventABI(configFilePath1, contract1)
	assert.Equal(t, err, ErrInvalidArguments)

	// valid arguments
	c, err := NewContractEventABI(configFilePath1, contract1, getX, getY)
	assert.NoError(t, err)
	topicEventNames := map[string]string{
		getXTopic: getX,
		getYTopic: getY,
	}
	assert.Equal(t, c.contract, contract1)
	assert.Equal(t, c.topicEventNames, topicEventNames)
}

var rs = `[{
	"contract": "0x",
	"failed": false,
	"logs": [
		{
			"address": "0x12fe58608430e36ba6bfb0a9bc5623a634530002",
			"data": [
				"0x0000000000000000000000000000000000000000000000000000000000000001",
				"0x0000000000000000000000000000000000000000000000000000000000000002"
			],
			"topic": "0x1086821eef716a909c39f2efe1e810bcd29246a6da19d04f9fc3f8d2889392e5"
		},
		{
			"address": "0x12fe58608430e36ba6bfb0a9bc5623a634530002",
			"data": [
				"0x0000000000000000000000000000000000000000000000000000000000000003",
				"0x0000000000000000000000000000000000000000000000000000000000000004"
			],
			"topic": "0x672e793f48f65acb771442258a567e553d1620c0684e1cbd9fe06ee380d1b642"
		}
	],
	"poststate": "0x67435ec564111d8bc235556727d2737c98d56a9a5f875005585790289e832dd5",
	"result": "0x0000000000000000000000000000000000000000000000000000000000000005",
	"totalFee": 243420,
	"txhash": "0xae073c03abc04ad182792bc5bf9faeb04d1c80888c985e839f896fd5fd08bf9f",
	"usedGas": 24342
},{
	"contract": "0x",
	"failed": false,
	"logs": [
		{
			"address": "0x170677801cb2a9faf387573c7fae61e440480002",
			"data": [
				"0x0000000000000000000000000000000000000000000000000000000000000020",
				"0x000000000000000000000000000000000000000000000000000000000000004e",
				"0x616263646566676869676b6c6d6e6f707172737475767778797a616263646566",
				"0x676869676b6c6d6e6f707172737475767778797a616263646566676869676b6c",
				"0x6d6e6f707172737475767778797a000000000000000000000000000000000000"
			],
			"topic": "0xa0acb9dd79e9d920ef642cb67cc5040eb54b29b163936c05777853bc5f4772b0"
		},
		{
			"address": "0x170677801cb2a9faf387573c7fae61e440480002",
			"topic": "0xa1c51915e437ec30e58312c6ff1ae0b5e7fc72426b83ddac06c2431e9edc5da1"
		}
	],
	"poststate": "0x13173cf873bc705d499da404ebb73f98b74427e1b117f8ba485a87f32e891a1a",
	"result": "0x0000000000000000000000000000000000000000000000000000000000000005",
	"totalFee": 246140,
	"txhash": "0x51c48e7eaabf25dba25f426487f559a25c3429e7d6ae57fae65ea262ca336e75",
	"usedGas": 24614
},{
	"contract": "0x",
	"failed": true,
	"logs": [
		{
			"address": "0x12fe58608430e36ba6bfb0a9bc5623a634530002",
			"data": [
				"0x0000000000000000000000000000000000000000000000000000000000000005",
				"0x0000000000000000000000000000000000000000000000000000000000000006"
			],
			"topic": "0x1086821eef716a909c39f2efe1e810bcd29246a6da19d04f9fc3f8d2889392e5"
		},
		{
			"address": "0x12fe58608430e36ba6bfb0a9bc5623a634530002",
			"data": [
				"0x0000000000000000000000000000000000000000000000000000000000000007",
				"0x0000000000000000000000000000000000000000000000000000000000000008"
			],
			"topic": "0x672e793f48f65acb771442258a567e553d1620c0684e1cbd9fe06ee380d1b642"
		}
	],
	"poststate": "0x67435ec564111d8bc235556727d2737c98d56a9a5f875005585790289e832dd5",
	"result": "0x0000000000000000000000000000000000000000000000000000000000000005",
	"totalFee": 243420,
	"txhash": "0xae073c03abc04ad182792bc5bf9faeb04d1c80888c985e839f896fd5fd08bf9f",
	"usedGas": 24342
}]`

func Test_ContractEventABI_ParseReceiptsToLogs(t *testing.T) {
	currentProjectPath, err := os.Getwd()
	assert.NoError(t, err)
	configFilePath1 := filepath.Join(currentProjectPath, path1)
	configFilePath2 := filepath.Join(currentProjectPath, path2)

	c1, err := NewContractEventABI(configFilePath1, contract1, getX, getY)
	assert.NoError(t, err)
	c2, err := NewContractEventABI(configFilePath2, contract2, getA, getB)
	assert.NoError(t, err)

	var receipts []*Receipt
	err = json.Unmarshal([]byte(rs), &receipts)
	assert.NoError(t, err)

	events1, err := c1.GetEventsFromReceipts(receipts)
	assert.NoError(t, err)
	assert.Equal(t, len(events1), 2)
	assert.Contains(t, events1, &Event{Contract: contract1, Topic: getXTopic, EventName: getX, Arguments: []interface{}{big.NewInt(3), big.NewInt(4)}})
	assert.Contains(t, events1, &Event{Contract: contract1, Topic: getYTopic, EventName: getY, Arguments: []interface{}{big.NewInt(1), big.NewInt(2)}})

	events2, err := c2.GetEventsFromReceipts(receipts)
	assert.NoError(t, err)
	assert.Equal(t, len(events2), 2)
	assert.Contains(t, events2, &Event{Contract: contract2, Topic: getATopic, EventName: getA, Arguments: []interface{}{argString}})
	assert.Contains(t, events2, &Event{Contract: contract2, Topic: getBTopic, EventName: getB})
}
