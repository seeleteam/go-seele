/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package seele

import (
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/stretchr/testify/assert"
)

func newTestSeeleBackend() *SeeleBackend {
	seeleService := newTestSeeleService()
	return &SeeleBackend{seeleService}
}

func Test_SeeleBackend_GetBlock(t *testing.T) {
	seeleBackend := newTestSeeleBackend()

	block, err := seeleBackend.GetBlock(common.EmptyHash, -1)
	assert.Equal(t, err, nil)
	assert.Equal(t, block.Header.Height, uint64(0))

	block1, err := seeleBackend.GetBlock(block.HeaderHash, -1)
	assert.Equal(t, err, nil)
	assert.Equal(t, block1.HeaderHash, block.HeaderHash)

	block2, err := seeleBackend.GetBlock(common.EmptyHash, 0)
	assert.Equal(t, err, nil)
	assert.Equal(t, block2.Header.Height, uint64(0))
	assert.Equal(t, block2.HeaderHash, block.HeaderHash)
}
