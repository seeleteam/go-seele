/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"math/big"
	"testing"
)

func Test_BigToDecimal(t *testing.T) {
	// 8位
	amount := big.NewInt(12345600)
	// 大于8位
	// amount := big.NewInt(123456789001234)
	// 小于8位
	// amount := big.NewInt(123456)
	v := amount.Int64()
	f := float64(v) / 100000000
	fmt.Println("v is:", v)
	fmt.Println("f is:", f)
	k := amount.Text(10)
	fmt.Println("k is:", k)
	length := len(k)
	if length > 8 {
		one := []byte(k[length-8 : length])
		two := []byte(k[:length-8])
		two = append(two, '.')
		two = append(two, one...)
		fmt.Println("the >8 number is:", string(two))
	} else if length == 8 {
		tag := []byte("0.")
		one := append(tag, k...)
		fmt.Println("the =8 number is:", string(one))
	} else {
		one := make([]byte, 10)
		one = append(one, "0."...)
		tag := []byte{'0', '0', '0', '0', '0', '0', '0', '0'}
		zero := tag[:8-length]
		one = append(one, zero...)
		one = append(one, k...)
		fmt.Println("the <8 number is:", string(one))

	}

}
