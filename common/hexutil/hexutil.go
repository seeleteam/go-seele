/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package hexutil

import (
	"encoding/hex"
)

var (
	// ErrEmptyString empty hex string
	ErrEmptyString = &decError{"empty hex string"}
	// ErrSyntax invalid hex string
	ErrSyntax = &decError{"invalid hex string"}
	// ErrMissingPrefix hex string without 0x prefix
	ErrMissingPrefix = &decError{"hex string without 0x prefix"}
	// ErrInvalidOddLength hex string of odd length
	ErrInvalidOddLength = &decError{"hex string of odd length"}
)

type decError struct{ msg string }

func (err *decError) Error() string { return err.msg }

// BytesToHex encodes b as a hex string with 0x prefix.
func BytesToHex(b []byte) string {
	enc := make([]byte, len(b)*2+2)
	copy(enc, "0x")
	hex.Encode(enc[2:], b)
	return string(enc)
}

// HexToBytes decodes a hex string with 0x prefix.
func HexToBytes(input string) ([]byte, error) {
	if len(input) == 0 {
		return nil, ErrEmptyString
	}

	// MissingPrefix
	if !Has0xPrefix(input) {
		return nil, ErrMissingPrefix
	}
	b, err := hex.DecodeString(input[2:])
	if err != nil {
		err = mapError(err)
	}
	return b, err
}

func MustHexToBytes(input string) []byte {
	result, err := HexToBytes(input)
	if err != nil {
		panic(err)
	}

	return result
}

// Has0xPrefix returns true if input start with 0x, otherwise false
func Has0xPrefix(input string) bool {
	return len(input) >= 2 && input[0] == '0' && (input[1] == 'x' || input[1] == 'X')
}

// mapError maps err to a more specific error
func mapError(err error) error {
	if _, ok := err.(hex.InvalidByteError); ok {
		return ErrSyntax
	}
	if err == hex.ErrLength {
		return ErrInvalidOddLength
	}
	return err
}
