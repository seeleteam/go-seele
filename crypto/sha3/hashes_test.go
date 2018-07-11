/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package sha3

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewKeccakAndSHA3(t *testing.T) {
	// NewKeccak256
	var result = NewKeccak256()
	assert.Equal(t, result.Size(), 32)
	assert.Equal(t, result.BlockSize(), 136)

	// NewKeccak512
	result = NewKeccak512()
	assert.Equal(t, result.Size(), 64)
	assert.Equal(t, result.BlockSize(), 72)

	// New224
	result = New224()
	assert.Equal(t, result.Size(), 28)
	assert.Equal(t, result.BlockSize(), 144)

	// New256
	result = New256()
	assert.Equal(t, result.Size(), 32)
	assert.Equal(t, result.BlockSize(), 136)

	// New384
	result = New384()
	assert.Equal(t, result.Size(), 48)
	assert.Equal(t, result.BlockSize(), 104)

	// New512
	result = New512()
	assert.Equal(t, result.Size(), 64)
	assert.Equal(t, result.BlockSize(), 72)
}

func Test_Sum(t *testing.T) {
	var bytes = []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

	// Sum224
	expectedResult224 := [28]byte{
		227, 140, 243, 55, 228, 140, 219, 75, 100, 167, 70, 185, 148, 190, 32, 120, 30, 81, 244, 84, 122, 76, 57, 36, 5, 153, 220, 186}
	assert.Equal(t, Sum224(bytes), expectedResult224)

	// Sum256
	expectedResult256 := [32]byte{
		96, 90, 5, 20, 5, 145, 146, 226, 109, 191, 6, 207, 171, 134, 243, 233, 187, 185, 166, 147, 99, 212, 190, 146, 91, 34, 70, 220, 216, 101, 154, 149}
	assert.Equal(t, Sum256(bytes), expectedResult256)

	// Sum384
	expectedResult384 := [48]byte{
		67, 210, 32, 131, 127, 85, 184, 160, 5, 140, 87, 40, 224, 37, 93, 192, 176, 7, 90, 108, 230, 157, 95, 202, 112, 75, 206, 92, 218, 225, 137, 99, 77,
		102, 11, 115, 109, 123, 199, 27, 50, 206, 218, 108, 204, 239, 142, 222}
	assert.Equal(t, Sum384(bytes), expectedResult384)

	// Sum512
	expectedResult512 := [64]byte{
		60, 82, 73, 212, 216, 20, 160, 236, 203, 50, 105, 245, 2, 79, 59, 207, 16, 39, 132, 224, 67, 160, 106, 106, 106, 155, 203, 253, 32, 247, 83, 187, 252,
		93, 160, 128, 142, 52, 248, 131, 109, 244, 40, 171, 159, 37, 65, 223, 236, 179, 109, 242, 136, 17, 44, 224, 38, 137, 2, 219, 229, 16, 89, 142}
	assert.Equal(t, Sum512(bytes), expectedResult512)
}
