package system

import (
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/stretchr/testify/assert"
)

func Test_GetContractByAddress(t *testing.T) {
	c := GetContractByAddress(domainNameContractAddress)
	assert.Equal(t, c, &contract{domainNameCommands})

	contractAddress := common.BytesToAddress([]byte{123, 1})
	c1 := GetContractByAddress(contractAddress)
	assert.Equal(t, c1, nil)
}
