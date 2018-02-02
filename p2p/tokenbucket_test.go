/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"fmt"
	"testing"
)

func Test_tokenBucket(t *testing.T) {
	tb := new(TokenBucket)
	tb.Init(1024*1024, 1.01)
	//curT := tb.GetCurTokens()
	tb.Consume(128 * 1024)
	fmt.Println("tokenBucket. cur=", tb.GetCurTokens())
}
