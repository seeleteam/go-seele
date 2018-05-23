/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

// CopyBytes copies and returns a new bytes from the specified source bytes.
func CopyBytes(src []byte) []byte {
	if src == nil {
		return nil
	}

	dest := make([]byte, len(src))
	copy(dest, src)
	return dest
}
