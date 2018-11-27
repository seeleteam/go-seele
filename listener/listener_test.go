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

	"github.com/seeleteam/go-seele/accounts/abi"
	"github.com/stretchr/testify/assert"
)

const path1 = `/testConfig/SimpleEventTest1.abi`
const getX = "getX"
const getY = "getY"

const path2 = `/testConfig/SimpleEventTest2.abi`
const getA = "getA"
const getB = "getB"

const getXTopic = "0x672e793f48f65acb771442258a567e553d1620c0684e1cbd9fe06ee380d1b642"
const getYTopic = "0x1086821eef716a909c39f2efe1e810bcd29246a6da19d04f9fc3f8d2889392e5"
const getATopic = "0xa0acb9dd79e9d920ef642cb67cc5040eb54b29b163936c05777853bc5f4772b0"
const getBTopic = "0xa1c51915e437ec30e58312c6ff1ae0b5e7fc72426b83ddac06c2431e9edc5da1"

const abi1 = `[
	{ "constant" : false, "inputs": [ { "name": "x", "type": "uint256" } ], "name": "set", "outputs": [], "payable": false, "stateMutability": "nonpayable", "type": "function" },
	{ "constant" : false, "inputs": [], "name": "get", "outputs": [ { "name": "", "type": "uint256" } ], "payable": false, "stateMutability": "nonpayable", "type": "function" },
	{ "inputs": [], "payable": false, "stateMutability": "nonpayable", "type": "constructor" },
	{ "anonymous": false, "inputs": [ { "indexed": false, "name": "", "type": "uint256" }, { "indexed": false, "name": "", "type": "uint256" } ], "name": "getX", "type": "event" },
	{ "anonymous": false, "inputs": [ { "indexed": false, "name": "", "type": "uint256" }, { "indexed": false, "name": "", "type": "uint256" } ], "name": "getY", "type": "event" }
]`

const abi2 = `[
	{ "constant": false, "inputs": [ { "name": "x", "type": "uint256" } ], "name": "set", "outputs": [], "payable": false, "stateMutability": "nonpayable", "type": "function" },
	{ "constant": false, "inputs": [], "name": "get", "outputs": [ { "name": "", "type": "uint256" } ], "payable": false, "stateMutability": "nonpayable", "type": "function" },
	{ "inputs": [], "payable": false, "stateMutability": "nonpayable", "type": "constructor" },
	{ "anonymous": false, "inputs": [ { "indexed": false, "name": "", "type": "string" } ], "name": "getA", "type": "event" },
	{ "anonymous": false, "inputs": [], "name": "getB", "type": "event" }
]`

const abierr = `
	{ "constant": false, "inputs": [ { "name": "x", "type": "uint256" } ], "name": "set", "outputs": [], "payable": false, "stateMutability": "nonpayable", "type": "function" },
	{ "constant": false, "inputs": [], "name": "get", "outputs": [ { "name": "", "type": "uint256" } ], "payable": false, "stateMutability": "nonpayable", "type": "function" },
	{ "inputs": [], "payable": false, "stateMutability": "nonpayable", "type": "constructor" },
	{ "anonymous": false, "inputs": [ { "indexed": false, "name": "", "type": "string" } ], "name": "getA", "type": "event" },
	{ "anonymous": false, "inputs": [], "name": "getB", "type": "event" }
]`

func newListener() (*Listener, error) {
	currentProjectPath, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	configFilePath1 := filepath.Join(currentProjectPath, path1)
	configFilePath2 := filepath.Join(currentProjectPath, path2)
	var inputs []string
	inputs = append(inputs, configFilePath1+","+getX+","+getY)
	inputs = append(inputs, configFilePath2+","+getA+","+getB)
	l, err := NewListener(inputs)
	if err != nil {
		return nil, err
	}

	return l, nil
}

func Test_NewListener(t *testing.T) {
	l, err := newListener()
	assert.NoError(t, err)
	assert.Equal(t, len(l.ABIParsers), 4)

	var abis []ABIInfo
	for key := range l.ABIParsers {
		abis = append(abis, key)
	}

	assert.Contains(
		t,
		abis,
		ABIInfo{
			ABI:       abi1,
			EventName: getX,
			Topic:     getXTopic,
		},
	)
	assert.Contains(
		t,
		abis,
		ABIInfo{
			ABI:       abi1,
			EventName: getY,
			Topic:     getYTopic,
		},
	)
	assert.Contains(
		t,
		abis,
		ABIInfo{
			ABI:       abi2,
			EventName: getA,
			Topic:     getATopic,
		},
	)
	assert.Contains(
		t,
		abis,
		ABIInfo{
			ABI:       abi2,
			EventName: getB,
			Topic:     getBTopic,
		},
	)
}

func Test_Listener_GetABIParsers_Event_ABI_Load_Failed(t *testing.T) {
	l := &Listener{
		ABIParsers: make(map[ABIInfo]abi.ABI),
	}
	cfg := &config{
		events: []abiEvent{
			abiEvent{
				abi:       abierr,
				eventName: getA,
			},
		},
	}
	err := l.getABIParsers(cfg)
	assert.Equal(t, err, ErrEventABILoadFailed)
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

func Test_Valid_Receipts_To_Events(t *testing.T) {
	l, err := newListener()
	assert.NoError(t, err)
	var receipts []*Receipt
	err = json.Unmarshal([]byte(rs), &receipts)
	assert.NoError(t, err)

	lg := GetGroupLogs(receipts)
	events, err := l.ParseLogGroupToEvents(lg)
	assert.NoError(t, err)
	assert.Equal(t, len(events), 4)
	assert.Contains(
		t,
		events,
		Event{
			At: ABIInfo{
				ABI:       abi1,
				EventName: getX,
				Topic:     getXTopic},
			Arguments: []interface{}{big.NewInt(3), big.NewInt(4)},
		},
	)
	assert.Contains(
		t,
		events,
		Event{
			At: ABIInfo{
				ABI:       abi1,
				EventName: getY,
				Topic:     getYTopic},
			Arguments: []interface{}{big.NewInt(1), big.NewInt(2)},
		},
	)
	assert.Contains(
		t,
		events,
		Event{
			At: ABIInfo{
				ABI:       abi2,
				EventName: getA,
				Topic:     getATopic},
			Arguments: []interface{}{"abcdefghigklmnopqrstuvwxyzabcdefghigklmnopqrstuvwxyzabcdefghigklmnopqrstuvwxyz"},
		},
	)
	assert.Contains(
		t,
		events,
		Event{
			At: ABIInfo{
				ABI:       abi2,
				EventName: getB,
				Topic:     getBTopic},
		},
	)
}

func Test_Empty_Receipts(t *testing.T) {
	l, err := newListener()
	assert.NoError(t, err)
	lg := GetGroupLogs(nil)
	events, err := l.ParseLogGroupToEvents(lg)
	assert.NoError(t, err)
	assert.Equal(t, len(events), 0)
}

func Test_Valid_Receipts_To_Events_By_ABIInfo(t *testing.T) {
	l, err := newListener()
	assert.NoError(t, err)
	abiInfo := ABIInfo{
		ABI:       abi1,
		EventName: getX,
		Topic:     getXTopic,
	}

	var receipts []*Receipt
	err = json.Unmarshal([]byte(rs), &receipts)
	assert.NoError(t, err)

	logs := GetLogsByTopic(getXTopic, receipts)
	events, err := l.ParseLogsToEventsByABIInfo(abiInfo, logs)
	assert.NoError(t, err)
	assert.Equal(t, len(events), 1)
	assert.Contains(
		t,
		events,
		Event{
			At: ABIInfo{ABI: abi1,
				EventName: getX,
				Topic:     getXTopic},
			Arguments: []interface{}{big.NewInt(3), big.NewInt(4)},
		},
	)
}

func Test_Listener_GetABIInfoByABIAndEventName(t *testing.T) {
	l, err := newListener()
	assert.NoError(t, err)
	var receipts []*Receipt
	err = json.Unmarshal([]byte(rs), &receipts)
	assert.NoError(t, err)

	lg := GetGroupLogs(receipts)
	events, err := l.ParseLogGroupToEvents(lg)
	assert.NoError(t, err)
	assert.Equal(t, len(events), 4)
	abi := abi1
	info, err := l.GetABIInfoByABIAndEventName(&abi, getX)
	assert.NoError(t, err)
	assert.Equal(
		t, ABIInfo{
			ABI:       abi1,
			EventName: getX,
			Topic:     getXTopic,
		},
		info)
}
