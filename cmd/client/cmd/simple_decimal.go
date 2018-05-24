/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package cmd

import (
	"fmt"
	"github.com/seeleteam/go-seele/miner/pow"
	"math/big"
	"strings"
)

//BigToDecimalfl simply changes big int to float64 which will miss 0 in the last
func BigToDecimalfl(amount *big.Int) string {
	base := pow.SeeleToCoin
	var quotient = big.NewInt(0)
	var mod = big.NewInt(0)
	quotient.Div(amount, base)
	mod.Mod(amount, base)
	numstr := quotient.Text(10) + "." + fmt.Sprintf("%08s", mod.Text(10))
	numstr = strings.TrimRight(numstr, "0")
	return numstr
}
