package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/seeleteam/go-seele/accounts/abi"
	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/stretchr/testify/assert"
)

func Test_printLogByABI(t *testing.T) {
	log := newTestLog(t)

	abiJSON := "[{\"constant\":true,\"inputs\":[],\"name\":\"creator\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"diceNumber\",\"type\":\"uint256\"},{\"name\":\"winValue\",\"type\":\"uint256\"}],\"name\":\"dice\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"destroy\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"senders\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"diceNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"randNumber\",\"type\":\"uint256\"}],\"name\":\"lossAction\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"diceNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"randNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"winValue\",\"type\":\"uint256\"}],\"name\":\"winAction\",\"type\":\"event\"}]"
	parsed, err1 := abi.JSON(strings.NewReader(abiJSON))
	assert.NoError(t, err1)

	logOut, err2 := printLogByABI(log, parsed)
	assert.NoError(t, err2)

	slog := &seeleLog{}
	err3 := json.Unmarshal([]byte(logOut), slog)
	assert.NoError(t, err3)
	assert.Equal(t, len(slog.Topics), len(log.Topics))
	assert.Equal(t, slog.Event, "lossAction")
	assert.Equal(t, len(slog.Args), 3)
	assert.Equal(t, slog.Args[0], "0xd3ee9ab572ed74f0b837ad9ea86f85e30e1dd6d1")
	assert.Equal(t, slog.Args[1], float64(50))
	assert.Equal(t, slog.Args[2], float64(60))
}

func newTestLog(t *testing.T) *types.Log {
	addr, err := common.HexToAddress("0xedf195e667b32d036f2a73aa37153068fd090012")
	assert.NoError(t, err)

	topic, err1 := common.HexToHash("0x655dc916988b3746402901627e8485408dccba300f8396bcc750826ca9a92182")
	assert.NoError(t, err1)
	topics := make([]common.Hash, 0)
	topics = append(topics, topic)

	log := &types.Log{
		Address: addr,
		Topics:  topics,
		Data:    []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 211, 238, 154, 181, 114, 237, 116, 240, 184, 55, 173, 158, 168, 111, 133, 227, 14, 29, 214, 209, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 50, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 60},
	}

	return log
}
