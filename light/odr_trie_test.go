/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package light

import (
	"testing"

	"github.com/seeleteam/go-seele/common"
	"github.com/stretchr/testify/assert"
)

func Test_odrTriePoof_Rlp(t *testing.T) {
	proof := odrTriePoof{
		Root:  common.StringToHash("root"),
		Key:   []byte("trie key"),
		Proof: make([]proofNode, 0),
	}

	encoded, err := common.Serialize(proof)
	assert.Nil(t, err)

	proof2 := odrTriePoof{}
	err = common.Deserialize(encoded, &proof2)
	assert.Nil(t, err)
	assert.Equal(t, proof, proof2)
}
