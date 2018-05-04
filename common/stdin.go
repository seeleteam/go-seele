/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"fmt"
	"github.com/howeyc/gopass"
	"bytes"
)

func GetPassword() (string, error) {
	fmt.Printf("Please input your key file password: ")
	pass, err := gopass.GetPasswd()
	if err != nil {
		return "", err
	}

	return string(pass), nil
}

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
		return "", fmt.Errorf("repeat password is not equal to orignal one")
	}

	return string(pass), nil
}
