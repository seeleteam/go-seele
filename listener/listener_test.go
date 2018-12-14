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

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/stretchr/testify/assert"
)

const path1 = `/testConfig/SimpleEventTest1.abi`
const getX = "getX"
const getY = "getY"

var getXTopic = common.MustHexToHash("0x672e793f48f65acb771442258a567e553d1620c0684e1cbd9fe06ee380d1b642")
var getYTopic = common.MustHexToHash("0x1086821eef716a909c39f2efe1e810bcd29246a6da19d04f9fc3f8d2889392e5")
var contract1 = common.HexMustToAddres("0x12fe58608430e36ba6bfb0a9bc5623a634530002")
var txHash1 = common.MustHexToHash("0xae073c03abc04ad182792bc5bf9faeb04d1c80888c985e839f896fd5fd08bf9f")

const path2 = `/testConfig/SimpleEventTest2.abi`
const getA = "getA"
const getB = "getB"

var getATopic = common.MustHexToHash("0xa0acb9dd79e9d920ef642cb67cc5040eb54b29b163936c05777853bc5f4772b0")
var getBTopic = common.MustHexToHash("0xa1c51915e437ec30e58312c6ff1ae0b5e7fc72426b83ddac06c2431e9edc5da1")
var contract2 = common.HexMustToAddres("0x170677801cb2a9faf387573c7fae61e440480002")
var txHash2 = common.MustHexToHash("0x51c48e7eaabf25dba25f426487f559a25c3429e7d6ae57fae65ea262ca336e75")

const argString = "abcdefghigklmnopqrstuvwxyzabcdefghigklmnopqrstuvwxyzabcdefghigklmnopqrstuvwxyz"

func Test_NewContractEventABI(t *testing.T) {
	currentProjectPath, err := os.Getwd()
	assert.NoError(t, err)
	configFilePath1 := filepath.Join(currentProjectPath, path1)

	// empty abi path
	_, err = NewContractEventABI("", contract1)
	assert.Equal(t, err, ErrInvalidArguments)

	// empty contract
	_, err = NewContractEventABI(configFilePath1, common.EmptyAddress, getX, getY)
	assert.Equal(t, err, ErrInvalidArguments)

	// empty events
	_, err = NewContractEventABI(configFilePath1, contract1)
	assert.Equal(t, err, ErrInvalidArguments)

	// valid arguments
	c, err := NewContractEventABI(configFilePath1, contract1, getX, getY)
	assert.NoError(t, err)
	topicEventNames := map[common.Hash]string{
		getXTopic: getX,
		getYTopic: getY,
	}
	assert.Equal(t, c.contract, contract1)
	assert.Equal(t, c.topicEventNames, topicEventNames)
}

var rs = `[{
	"ContractAddress": "",
	"Failed": false,
	"Logs": [
		{
			"address": "0x12fe58608430e36ba6bfb0a9bc5623a634530002",
			"blockNumber": 112,
			"data": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAg==",
			"topics": [
				"0x672e793f48f65acb771442258a567e553d1620c0684e1cbd9fe06ee380d1b642"
			],
			"transactionIndex": 1
		},
		{
			"address": "0x12fe58608430e36ba6bfb0a9bc5623a634530002",
			"blockNumber": 112,
			"data": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAMAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABA==",
			"topics": [
				"0x1086821eef716a909c39f2efe1e810bcd29246a6da19d04f9fc3f8d2889392e5"
			],
			"transactionIndex": 1
		}
	],
	"PostState": "0xa0e88398aa7a0a84d8ef852ff11ca377fac25931841aff76b30e0b6684182783",
	"Result": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAU=",
	"TotalFee": 243420,
	"TxHash": "0xae073c03abc04ad182792bc5bf9faeb04d1c80888c985e839f896fd5fd08bf9f",
	"UsedGas": 24342
},{
	"ContractAddress": "",
	"Failed": false,
	"Logs": [
		{
			"address": "0x170677801cb2a9faf387573c7fae61e440480002",
			"blockNumber": 1062,
			"data": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAATmFiY2RlZmdoaWdrbG1ub3BxcnN0dXZ3eHl6YWJjZGVmZ2hpZ2tsbW5vcHFyc3R1dnd4eXphYmNkZWZnaGlna2xtbm9wcXJzdHV2d3h5egAAAAAAAAAAAAAAAAAAAAAAAA==",
			"topics": [
				"0xa0acb9dd79e9d920ef642cb67cc5040eb54b29b163936c05777853bc5f4772b0"
			],
			"transactionIndex": 1
		},
		{
			"address": "0x170677801cb2a9faf387573c7fae61e440480002",
			"blockNumber": 1062,
			"data": "",
			"topics": [
				"0xa1c51915e437ec30e58312c6ff1ae0b5e7fc72426b83ddac06c2431e9edc5da1"
			],
			"transactionIndex": 1
		}
	],
	"PostState": "0x13173cf873bc705d499da404ebb73f98b74427e1b117f8ba485a87f32e891a1a",
	"Result": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAU=",
	"TotalFee": 246140,
	"TxHash": "0x51c48e7eaabf25dba25f426487f559a25c3429e7d6ae57fae65ea262ca336e75",
	"UsedGas": 24614
},{
	"ContractAddress": "",
	"Failed": true,
	"Logs": [
		{
			"address": "0x12fe58608430e36ba6bfb0a9bc5623a634530002",
			"blockNumber": 112,
			"data": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAg==",
			"topics": [
				"0x672e793f48f65acb771442258a567e553d1620c0684e1cbd9fe06ee380d1b642"
			],
			"transactionIndex": 1
		},
		{
			"address": "0x12fe58608430e36ba6bfb0a9bc5623a634530002",
			"blockNumber": 112,
			"data": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAMAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABA==",
			"topics": [
				"0x1086821eef716a909c39f2efe1e810bcd29246a6da19d04f9fc3f8d2889392e5"
			],
			"transactionIndex": 1
		}
	],
	"PostState": "0xa0e88398aa7a0a84d8ef852ff11ca377fac25931841aff76b30e0b6684182783",
	"Result": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAU=",
	"TotalFee": 243420,
	"TxHash": "0xae073c03abc04ad182792bc5bf9faeb04d1c80888c985e839f896fd5fd08bf9f",
	"UsedGas": 24342
}]`

func Test_ContractEventABI_GetEvent(t *testing.T) {
	currentProjectPath, err := os.Getwd()
	assert.NoError(t, err)
	configFilePath1 := filepath.Join(currentProjectPath, path1)

	c1, err := NewContractEventABI(configFilePath1, contract1, getX, getY)
	assert.NoError(t, err)

	var receipts []*types.Receipt
	err = json.Unmarshal([]byte(rs), &receipts)
	assert.NoError(t, err)

	events1, err := c1.GetEvent(receipts[0])
	assert.NoError(t, err)
	assert.Equal(t, len(events1), 2)
	assert.Contains(t, events1, &Event{TxHash: txHash1, Contract: contract1, EventName: getX, Topic: getXTopic, Arguments: []interface{}{big.NewInt(1), big.NewInt(2)}})
	assert.Contains(t, events1, &Event{TxHash: txHash1, Contract: contract1, EventName: getY, Topic: getYTopic, Arguments: []interface{}{big.NewInt(3), big.NewInt(4)}})
}

func Test_ContractEventABI_GetEvents(t *testing.T) {
	currentProjectPath, err := os.Getwd()
	assert.NoError(t, err)
	configFilePath1 := filepath.Join(currentProjectPath, path1)
	configFilePath2 := filepath.Join(currentProjectPath, path2)

	c1, err := NewContractEventABI(configFilePath1, contract1, getX, getY)
	assert.NoError(t, err)
	c2, err := NewContractEventABI(configFilePath2, contract2, getA, getB)
	assert.NoError(t, err)

	var receipts []*types.Receipt
	err = json.Unmarshal([]byte(rs), &receipts)
	assert.NoError(t, err)

	events1, err := c1.GetEvents(receipts)
	assert.NoError(t, err)
	assert.Equal(t, len(events1), 2)
	assert.Contains(t, events1, &Event{TxHash: txHash1, Contract: contract1, EventName: getX, Topic: getXTopic, Arguments: []interface{}{big.NewInt(1), big.NewInt(2)}})
	assert.Contains(t, events1, &Event{TxHash: txHash1, Contract: contract1, EventName: getY, Topic: getYTopic, Arguments: []interface{}{big.NewInt(3), big.NewInt(4)}})

	events2, err := c2.GetEvents(receipts)
	assert.NoError(t, err)
	assert.Equal(t, len(events2), 2)
	assert.Contains(t, events2, &Event{TxHash: txHash2, Contract: contract2, EventName: getA, Topic: getATopic, Arguments: []interface{}{argString}})
	assert.Contains(t, events2, &Event{TxHash: txHash2, Contract: contract2, EventName: getB, Topic: getBTopic, Arguments: []interface{}{}})
}
