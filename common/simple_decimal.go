/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package common

import (
	"fmt"
	"math/big"
	"strings"
)

var (
	//SeeleToCoin base coin number
	SeeleToCoin = big.NewInt(100000000)
)

//BigToDecimal simply changes big int to decimal which will miss additional 0 in the last
func BigToDecimal(amount *big.Int) string {
	base := SeeleToCoin
	var quotient = big.NewInt(0)
	var mod = big.NewInt(0)
	var numstr string
	quotient.Div(amount, base)
	mod.Mod(amount, base)
	modValue := mod.Text(10)
	quotientValue := quotient.Text(10)
	if strings.EqualFold(modValue, "0") {
		numstr = quotientValue
	} else {
		numstr = quotientValue + "." + fmt.Sprintf("%08s", modValue)
		numstr = strings.TrimRight(numstr, "0")
	}
	return numstr
}
