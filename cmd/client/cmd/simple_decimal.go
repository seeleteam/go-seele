/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"math/big"
)

const (
	base = 100000000
	num  = 8
)

//BigToDecimalStr big int changes to decimal string number with no missing number 0
func BigToDecimalStr(amount *big.Int) string {
	k := amount.Text(10)
	length := len(k)
	var numstring string
	if length > num {
		one := []byte(k[length-num : length])
		two := []byte(k[:length-num])
		two = append(two, '.')
		two = append(two, one...)
		numstring = string(two)
	} else if length == num {
		tag := []byte("0.")
		one := append(tag, k...)
		numstring = string(one)
	} else {
		one := make([]byte, num+2)
		one = append(one, "0."...)
		tag := []byte{'0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0', '0'}
		zero := tag[:num-length]
		one = append(one, zero...)
		one = append(one, k...)
		numstring = string(one)
	}
	return numstring
}

//BigToDecimalfl simply changes big int to float64 which will miss 0 in the last
func BigToDecimalfl(amount *big.Int) float64 {
	v := amount.Int64()
	f := float64(v) / base
	return f
}
