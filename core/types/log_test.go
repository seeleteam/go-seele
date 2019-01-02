package types

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/seeleteam/go-seele/common/hexutil"

	"github.com/seeleteam/go-seele/common"
	"github.com/stretchr/testify/assert"
)

func Test_MarshalJSON(t *testing.T) {
	count, addrHex, hash, dataHex := 5, "0x6d05ccde7e91439e0de160335ee87a9a219c0002", common.BytesToHash([]byte("asdf")), "0x000000000000000000000000f4c5625a3a193c5261e7d26de446f86c1c2d2561000000000000000000000000f4c5625a3a193c5261e7d26de446f86c1c2d2561"

	address, err := common.HexToAddress(addrHex)
	assert.NoError(t, err)

	topics := make([]common.Hash, count)
	for index := 0; index < count; index++ {
		topics[index] = hash
	}

	data, err := hexutil.HexToBytes(dataHex)
	assert.NoError(t, err)
	log := &Log{
		Address:     address,
		Topics:      topics,
		Data:        data,
		BlockNumber: 65484,
		TxIndex:     2,
	}

	encoded, err := json.Marshal(log)
	assert.NoError(t, err)
	assert.NotEmpty(t, encoded)

	str := string(encoded)
	assert.True(t, strings.Contains(str, "address"))
	assert.True(t, strings.Contains(str, "topics"))
	assert.True(t, strings.Contains(str, "data"))
	assert.True(t, strings.Contains(str, "blockNumber"))
	assert.True(t, strings.Contains(str, "transactionIndex"))
	assert.True(t, strings.Contains(str, addrHex))
	assert.True(t, strings.Contains(str, hash.Hex()))
	assert.True(t, strings.Contains(str, dataHex))
}
