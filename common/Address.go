/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

const (
	HashLength = 32
)

type Hash [HashLength]byte

func BytesToAddress(b []byte) Hash {
	var a Hash
	a.SetBytes(b)
	return a
}

func StringToAddress(s string) Hash {
	return BytesToAddress([]byte(s))
}

// Sets the hash to the value of b. If b is larger than len(a) it will panic
func (a *Hash) SetBytes(b []byte) {
	copy(a[:], b)
}
