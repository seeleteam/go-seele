package svm

import (
	"math/big"
	"testing"

	"github.com/seeleteam/go-seele/core/svm/evm"
	"github.com/seeleteam/go-seele/core/svm/native"
	"github.com/stretchr/testify/assert"
)

func Test_CreateSVM(t *testing.T) {
	ctx := newTestContext(t, big.NewInt(0))

	// EVM
	svm := CreateSVM(ctx, EVM)
	_, ok := svm.(*evm.EVM)
	assert.Equal(t, ok, true)

	// Native
	svm1 := CreateSVM(ctx, Native)
	_, ok1 := svm1.(*native.NVM)
	assert.Equal(t, ok1, true)

	// Other
	var typ Type = 354
	svm2 := CreateSVM(ctx, typ)
	_, ok2 := svm2.(*evm.EVM)
	assert.Equal(t, ok2, true)
}
