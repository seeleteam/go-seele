/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"bytes"
	"fmt"

	"github.com/howeyc/gopass"
	"github.com/seeleteam/go-seele/common/errors"
)

// GetPassword ask user for password interactively
func GetPassword() (string, error) {
	fmt.Printf("Please input your key file password: ")
	pass, err := gopass.GetPasswd()
	if err != nil {
		return "", err
	}

	return string(pass), nil
}

// SetPassword ask user input password twice and get the password interactively
func SetPassword() (string, error) {
	fmt.Printf("Password: ")
	pass, err := gopass.GetPasswd()
	if err != nil {
		return "", err
	}

	fmt.Printf("Repeat password:")
	passRepeat, err := gopass.GetPasswd()
	if err != nil {
		return "", err
	}

	if !bytes.Equal(pass, passRepeat) {
		return "", errors.Get(errors.ErrPasswordRepeatMismatch)
	}

	return string(pass), nil
}
