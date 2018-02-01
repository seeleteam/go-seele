/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package p2p

import (
	"fmt"
	"testing"
)

func Test_test1(t *testing.T) {
	tb := new(TokenBucket)
	tb.Init(1024*1024, 1.01)
	fmt.Println("hello")
}
