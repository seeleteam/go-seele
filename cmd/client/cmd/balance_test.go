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
	amount := big.NewInt(9876123456000)
	// 大于8位
	// amount := big.NewInt(123456789001234)
	// 小于8位
	// amount := big.NewInt(123456)
	base := big.NewInt(100000000)
	var quotient = big.NewInt(0)
	var mod = big.NewInt(0)
	quotient.Div(amount, base)
	mod.Mod(amount, base)
	numstr := quotient.Text(10) + "." + fmt.Sprintf("%08s", mod.Text(10))
	numstr = strings.TrimRight(numstr, "0")
	fmt.Println("amount is:", amount)
	fmt.Println("Base is:", base)
	fmt.Println("Quotient is:", quotient)
	fmt.Println("Mod is:", mod)
	fmt.Println("numstr is:", numstr)

}
