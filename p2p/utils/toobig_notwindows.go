/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

//+build !windows

package utils

// isPacketTooBig reports whether err indicates that a UDP packet didn't
// fit the receive buffer. There is no such error on
// non-Windows platforms.
func isPacketTooBig(err error) bool {
	return false
}
