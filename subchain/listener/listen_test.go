/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package listener

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

const TestEvent1 = "getX"
const TestABI1 = `
[
	{ "constant" : false, "inputs": [ { "name": "x", "type": "uint256" } ], "name": "set", "outputs": [], "payable": false, "stateMutability": "nonpayable", "type": "function" },
	{ "constant" : false, "inputs": [], "name": "get", "outputs": [ { "name": "", "type": "uint256" } ], "payable": false, "stateMutability": "nonpayable", "type": "function" },
	{ "inputs": [], "payable": false, "stateMutability": "nonpayable", "type": "constructor" },
	{ "anonymous": false, "inputs": [ { "indexed": false, "name": "", "type": "uint256" }, { "indexed": false, "name": "", "type": "uint256" } ], "name": "getX", "type": "event" },
	{ "anonymous": false, "inputs": [ { "indexed": false, "name": "", "type": "uint256" }, { "indexed": false, "name": "", "type": "uint256" } ], "name": "getY", "type": "event" }
]`

const TestEvent2 = "getY"
const TestABI2 = `
[
	{ "constant" : false, "inputs": [ { "name": "x", "type": "uint256" } ], "name": "set", "outputs": [], "payable": false, "stateMutability": "nonpayable", "type": "function" },
	{ "constant" : false, "inputs": [], "name": "get", "outputs": [ { "name": "", "type": "uint256" } ], "payable": false, "stateMutability": "nonpayable", "type": "function" },
	{ "inputs": [], "payable": false, "stateMutability": "nonpayable", "type": "constructor" },
	{ "anonymous": false, "inputs": [ { "indexed": false, "name": "", "type": "uint256" }, { "indexed": false, "name": "", "type": "uint256" } ], "name": "getX", "type": "event" },
	{ "anonymous": false, "inputs": [ { "indexed": false, "name": "", "type": "uint256" }, { "indexed": false, "name": "", "type": "uint256" } ], "name": "getY", "type": "event" }
]`

var testRefMp = map[string]string{
	TestEvent1: TestABI1,
	TestEvent2: TestABI2,
}

func TestListener_GetABIParser_Empty_Config(t *testing.T) {
	l := NewListener(nil)
	l.cfg = nil
	err := l.GetABIParser()
	assert.Equal(t, err, ErrEventCfgEmpty)
}

func TestListener_GetABIParser_Event_API_Empty(t *testing.T) {
	l := NewListener(testRefMp)
	l.cfg.events[0].abi = ""
	err := l.GetABIParser()
	assert.Equal(t, err, ErrEventABIEmpty)
}

func TestListener_GetABIParser_Event_ABI_Load_Failed(t *testing.T) {
	l := NewListener(testRefMp)
	l.cfg.events[0].abi = l.cfg.events[0].abi[2:]
	err := l.GetABIParser()
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
			"topic": "0x672e793f48f65acb771442258a567e553d1620c0684e1cbd9fe06ee380d1b642"
		},
		{
			"address": "0x12fe58608430e36ba6bfb0a9bc5623a634530002",
			"data": [
				"0x0000000000000000000000000000000000000000000000000000000000000003",
				"0x0000000000000000000000000000000000000000000000000000000000000004"
			],
			"topic": "0x1086821eef716a909c39f2efe1e810bcd29246a6da19d04f9fc3f8d2889392e5"
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
			"address": "0x12fe58608430e36ba6bfb0a9bc5623a634530002",
			"data": [
				"0x0000000000000000000000000000000000000000000000000000000000000005",
				"0x0000000000000000000000000000000000000000000000000000000000000006"
			],
			"topic": "0x672e793f48f65acb771442258a567e553d1620c0684e1cbd9fe06ee380d1b642"
		},
		{
			"address": "0x12fe58608430e36ba6bfb0a9bc5623a634530002",
			"data": [
				"0x0000000000000000000000000000000000000000000000000000000000000007",
				"0x0000000000000000000000000000000000000000000000000000000000000008"
			],
			"topic": "0x1086821eef716a909c39f2efe1e810bcd29246a6da19d04f9fc3f8d2889392e5"
		}
	],
	"poststate": "0x67435ec564111d8bc235556727d2737c98d56a9a5f875005585790289e832dd5",
	"result": "0x0000000000000000000000000000000000000000000000000000000000000005",
	"totalFee": 243420,
	"txhash": "0xae073c03abc04ad182792bc5bf9faeb04d1c80888c985e839f896fd5fd08bf9f",
	"usedGas": 24342
}]`

func Test_No_Failed_Receipts(t *testing.T) {
	l := NewListener(testRefMp)
	var receipts []*Receipt
	err := json.Unmarshal([]byte(rs), &receipts)
	assert.Equal(t, err, nil)

	err = l.GetABIParser()
	assert.Equal(t, err, nil)

	lg := GroupByTopic(receipts)
	events := l.Filter(lg)
	assert.Equal(t, len(events), 4)
	assert.Contains(
		t,
		events,
		Event{MethodName: "getX", Topic: "0x672e793f48f65acb771442258a567e553d1620c0684e1cbd9fe06ee380d1b642", Arguments: []interface{}{big.NewInt(1), big.NewInt(2)}},
	)
	assert.Contains(
		t,
		events,
		Event{MethodName: "getX", Topic: "0x672e793f48f65acb771442258a567e553d1620c0684e1cbd9fe06ee380d1b642", Arguments: []interface{}{big.NewInt(5), big.NewInt(6)}},
	)
	assert.Contains(
		t,
		events,
		Event{MethodName: "getY", Topic: "0x1086821eef716a909c39f2efe1e810bcd29246a6da19d04f9fc3f8d2889392e5", Arguments: []interface{}{big.NewInt(3), big.NewInt(4)}},
	)
	assert.Contains(
		t,
		events,
		Event{MethodName: "getY", Topic: "0x1086821eef716a909c39f2efe1e810bcd29246a6da19d04f9fc3f8d2889392e5", Arguments: []interface{}{big.NewInt(7), big.NewInt(8)}},
	)
}

func Test_Empty_Receipts(t *testing.T) {
	l := NewListener(testRefMp)
	err := l.GetABIParser()
	assert.Equal(t, err, nil)

	lg := GroupByTopic(nil)
	events := l.Filter(lg)
	var es []Event
	assert.Equal(t, events, es)
}

var rs1 = `[{
	"contract": "0x",
	"failed": false,
	"logs": [
		{
			"address": "0x12fe58608430e36ba6bfb0a9bc5623a634530002",
			"data": [
				"0x0000000000000000000000000000000000000000000000000000000000000001",
				"0x0000000000000000000000000000000000000000000000000000000000000002"
			],
			"topic": "0x672e793f48f65acb771442258a567e553d1620c0684e1cbd9fe06ee380d1b642"
		},
		{
			"address": "0x12fe58608430e36ba6bfb0a9bc5623a634530002",
			"data": [
				"0x0000000000000000000000000000000000000000000000000000000000000003",
				"0x0000000000000000000000000000000000000000000000000000000000000004"
			],
			"topic": "0x1086821eef716a909c39f2efe1e810bcd29246a6da19d04f9fc3f8d2889392e5"
		}
	],
	"poststate": "0x67435ec564111d8bc235556727d2737c98d56a9a5f875005585790289e832dd5",
	"result": "0x0000000000000000000000000000000000000000000000000000000000000005",
	"totalFee": 243420,
	"txhash": "0xae073c03abc04ad182792bc5bf9faeb04d1c80888c985e839f896fd5fd08bf9f",
	"usedGas": 24342
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
			"topic": "0x672e793f48f65acb771442258a567e553d1620c0684e1cbd9fe06ee380d1b642"
		},
		{
			"address": "0x12fe58608430e36ba6bfb0a9bc5623a634530002",
			"data": [
				"0x0000000000000000000000000000000000000000000000000000000000000007",
				"0x0000000000000000000000000000000000000000000000000000000000000008"
			],
			"topic": "0x1086821eef716a909c39f2efe1e810bcd29246a6da19d04f9fc3f8d2889392e5"
		}
	],
	"poststate": "0x67435ec564111d8bc235556727d2737c98d56a9a5f875005585790289e832dd5",
	"result": "0x0000000000000000000000000000000000000000000000000000000000000005",
	"totalFee": 243420,
	"txhash": "0xae073c03abc04ad182792bc5bf9faeb04d1c80888c985e839f896fd5fd08bf9f",
	"usedGas": 24342
}]`

func Test_Failed_Receipts(t *testing.T) {
	l := NewListener(testRefMp)
	var receipts []*Receipt
	err := json.Unmarshal([]byte(rs1), &receipts)
	assert.Equal(t, err, nil)

	err = l.GetABIParser()
	assert.Equal(t, err, nil)

	lg := GroupByTopic(receipts)
	events := l.Filter(lg)
	assert.Equal(t, len(events), 2)
	assert.Contains(
		t,
		events,
		Event{MethodName: "getX", Topic: "0x672e793f48f65acb771442258a567e553d1620c0684e1cbd9fe06ee380d1b642", Arguments: []interface{}{big.NewInt(1), big.NewInt(2)}},
	)
	assert.Contains(
		t,
		events,
		Event{MethodName: "getY", Topic: "0x1086821eef716a909c39f2efe1e810bcd29246a6da19d04f9fc3f8d2889392e5", Arguments: []interface{}{big.NewInt(3), big.NewInt(4)}},
	)
}

var rs2 = `[{
	"contract": "0x",
	"failed": false,
	"logs": [
		{
			"address": "0x12fe58608430e36ba6bfb0a9bc5623a634530002",
			"topic": "0x672e793f48f65acb771442258a567e553d1620c0684e1cbd9fe06ee380d1b642"
		},
		{
			"address": "0x12fe58608430e36ba6bfb0a9bc5623a634530002",
			"data": [
				"0x0000000000000000000000000000000000000000000000000000000000000003",
				"0x0000000000000000000000000000000000000000000000000000000000000004"
			],
			"topic": "0x1086821eef716a909c39f2efe1e810bcd29246a6da19d04f9fc3f8d2889392e5"
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
			"address": "0x12fe58608430e36ba6bfb0a9bc5623a634530002",
			"data": [
				"0x0000000000000000000000000000000000000000000000000000000000000005",
				"0x0000000000000000000000000000000000000000000000000000000000000006"
			],
			"topic": "0x672e793f48f65acb771442258a567e553d1620c0684e1cbd9fe06ee380d1b642"
		},
		{
			"address": "0x12fe58608430e36ba6bfb0a9bc5623a634530002",
			"topic": "0x1086821eef716a909c39f2efe1e810bcd29246a6da19d04f9fc3f8d2889392e5"
		}
	],
	"poststate": "0x67435ec564111d8bc235556727d2737c98d56a9a5f875005585790289e832dd5",
	"result": "0x0000000000000000000000000000000000000000000000000000000000000005",
	"totalFee": 243420,
	"txhash": "0xae073c03abc04ad182792bc5bf9faeb04d1c80888c985e839f896fd5fd08bf9f",
	"usedGas": 24342
}]`

func Test_Empty_Data_Receipts_Log(t *testing.T) {
	l := NewListener(testRefMp)
	var receipts []*Receipt
	err := json.Unmarshal([]byte(rs2), &receipts)
	assert.Equal(t, err, nil)

	err = l.GetABIParser()
	assert.Equal(t, err, nil)

	lg := GroupByTopic(receipts)
	events := l.Filter(lg)
	assert.Equal(t, len(events), 4)
	assert.Contains(
		t,
		events,
		Event{MethodName: "getX", Topic: "0x672e793f48f65acb771442258a567e553d1620c0684e1cbd9fe06ee380d1b642"},
	)
	assert.Contains(
		t,
		events,
		Event{MethodName: "getX", Topic: "0x672e793f48f65acb771442258a567e553d1620c0684e1cbd9fe06ee380d1b642", Arguments: []interface{}{big.NewInt(5), big.NewInt(6)}},
	)
	assert.Contains(
		t,
		events,
		Event{MethodName: "getY", Topic: "0x1086821eef716a909c39f2efe1e810bcd29246a6da19d04f9fc3f8d2889392e5", Arguments: []interface{}{big.NewInt(3), big.NewInt(4)}},
	)
	assert.Contains(
		t,
		events,
		Event{MethodName: "getY", Topic: "0x1086821eef716a909c39f2efe1e810bcd29246a6da19d04f9fc3f8d2889392e5"},
	)
}
