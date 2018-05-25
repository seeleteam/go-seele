/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func Test_BigToDecimal(t *testing.T) {
	// 8位
	amount := big.NewInt(1234567)
	base := big.NewInt(100000000)
	var quotient = big.NewInt(0)
	var mod = big.NewInt(0)
	var numstr string
	quotient.Div(amount, base)
	mod.Mod(amount, base)
	modValue := mod.Text(10)
	if strings.EqualFold(modValue, "0") {
		numstr = quotient.Text(10)
	} else {
		numstr = quotient.Text(10) + "." + fmt.Sprintf("%08s", modValue)
		numstr = strings.TrimRight(numstr, "0")
	}

	fmt.Println("amount is:", amount)
	fmt.Println("Base is:", base)
	fmt.Println("Quotient is:", quotient)
	fmt.Println("Mod is:", mod)
	fmt.Println("numstr is:", numstr)
	// 大于8位
	amount = big.NewInt(100000000)
	quotient.Div(amount, base)
	mod.Mod(amount, base)
	modValue = mod.Text(10)
	if strings.EqualFold(modValue, "0") {
		numstr = quotient.Text(10)
	} else {
		numstr = quotient.Text(10) + "." + fmt.Sprintf("%08s", modValue)
		numstr = strings.TrimRight(numstr, "0")
	}
	fmt.Println("amount is:", amount)
	fmt.Println("Base is:", base)
	fmt.Println("Quotient is:", quotient)
	fmt.Println("Mod is:", mod)
	fmt.Println("numstr is:", numstr)

	// 大于8位
	amount = big.NewInt(1000000001234)
	quotient.Div(amount, base)
	mod.Mod(amount, base)
	modValue = mod.Text(10)
	if strings.EqualFold(modValue, "0") {
		numstr = quotient.Text(10)
	} else {
		numstr = quotient.Text(10) + "." + fmt.Sprintf("%08s", modValue)
		numstr = strings.TrimRight(numstr, "0")
	}
	fmt.Println("amount is:", amount)
	fmt.Println("Base is:", base)
	fmt.Println("Quotient is:", quotient)
	fmt.Println("Mod is:", mod)
	fmt.Println("numstr is:", numstr)
	// 大于8位
	amount = big.NewInt(112300023000)
	quotient.Div(amount, base)
	mod.Mod(amount, base)
	modValue = mod.Text(10)
	if strings.EqualFold(modValue, "0") {
		numstr = quotient.Text(10)
	} else {
		numstr = quotient.Text(10) + "." + fmt.Sprintf("%08s", modValue)
		numstr = strings.TrimRight(numstr, "0")
	}
	fmt.Println("amount is:", amount)
	fmt.Println("Base is:", base)
	fmt.Println("Quotient is:", quotient)
	fmt.Println("Mod is:", mod)
	fmt.Println("numstr is:", numstr)
	// 小于8位
	amount = big.NewInt(0)
	quotient.Div(amount, base)
	mod.Mod(amount, base)
	modValue = mod.Text(10)
	if strings.EqualFold(modValue, "0") {
		numstr = quotient.Text(10)
	} else {
		numstr = quotient.Text(10) + "." + fmt.Sprintf("%08s", modValue)
		numstr = strings.TrimRight(numstr, "0")
	}
	fmt.Println("amount is:", amount)
	fmt.Println("Base is:", base)
	fmt.Println("Quotient is:", quotient)
	fmt.Println("Mod is:", mod)
	fmt.Println("numstr is:", numstr)

}
