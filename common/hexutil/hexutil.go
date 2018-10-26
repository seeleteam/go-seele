/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package hexutil

import (
	"encoding/hex"
)

const (
	// AddressLen is the valid address length
	AddressLen = 20
)

var (
	// ErrEmptyString empty hex string
	ErrEmptyString = &decError{"empty hex string"}
	// ErrSyntax invalid hex string
	ErrSyntax = &decError{"invalid hex string"}
	// ErrMissingPrefix hex string without 0x prefix
	ErrMissingPrefix = &decError{"hex string without 0x prefix"}
	// ErrOddLength hex string of odd length
	ErrOddLength = &decError{"hex string of odd length"}
	// ErrInvalidLength hex string's length must be  equal or gratter than 20
	ErrInvalidLength = &decError{"hex string's length must be  equal or gratter than 20"}
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
func HexToBytes(inputs ...string) ([]byte, error) {
	input := inputs[0]
	var flag string
	if len(inputs) >= 2 {
		flag = inputs[1]
	}
	if len(input) == 0 {
		return nil, ErrEmptyString
	}
	if flag == "client" { //if the request come from client command
		//length less 20 is valid length
		if len(input) < AddressLen {
			return nil, ErrInvalidLength
		}
	}
	//MissingPrefix
	if !Has0xPrefix(input) {
		return nil, ErrMissingPrefix
	}
	b, err := hex.DecodeString(input[2:])
	if err != nil {
		err = mapError(err)
	}
	return b, err
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
		return ErrOddLength
	}
	return err
}
