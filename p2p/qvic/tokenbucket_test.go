/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package qvic

import (
	"fmt"
	"testing"
	"time"
)

func Test_TokenBucketInitAndAdjust(t *testing.T) {
	var bytesPerSecond int64 = 1024
	v1 := new(TokenBucket)
	v1.Init(bytesPerSecond)

	if v1.bytesPerSecond != bytesPerSecond || v1.maxTokens != bytesPerSecond || v1.tokensPerMS != bytesPerSecond/1000 ||
		v1.reservedTokens != bytesPerSecond/10 || v1.curTokens != bytesPerSecond || v1.preTick < 0 {
		fmt.Println("TokenBucket Init", v1.bytesPerSecond, v1.maxTokens, v1.tokensPerMS, v1.reservedTokens, v1.curTokens, v1.preTick)
		t.Fail()
	}

	newBytesPerSecond := bytesPerSecond + 1
	v1.AdjustBW(newBytesPerSecond)
	if v1.bytesPerSecond != newBytesPerSecond || v1.maxTokens != newBytesPerSecond || v1.tokensPerMS != newBytesPerSecond/1000 ||
		v1.reservedTokens != newBytesPerSecond/10 || v1.curTokens != bytesPerSecond || v1.preTick < 0 {
		fmt.Println("TokenBucket Adjust", v1.bytesPerSecond, v1.maxTokens, v1.tokensPerMS, v1.reservedTokens, v1.curTokens, v1.preTick)
		t.Fail()
	}
}

func Test_TokenBucketConsumeAndPeriodicFeed(t *testing.T) {
	var bytesPerSecond int64 = 1000
	v1 := new(TokenBucket)
	v1.Init(bytesPerSecond)

	// Initially curTokens must be equal to v1.maxTokens
	if v1.curTokens != v1.maxTokens {
		fmt.Println("ConsumeAndPeriodicFeed init:", v1.maxTokens, v1.curTokens)
		t.Fail()
	}

	// Feed it
	time.Sleep(100 * time.Millisecond)
	v1.PeriodicFeed()
	if v1.curTokens != v1.maxTokens {
		fmt.Println("ConsumeAndPeriodicFeed feed:", v1.maxTokens, v1.curTokens)
		t.Fail()
	}

	// Use all of tokens
	v1.Consume(1000)
	if v1.curTokens != 0 {
		fmt.Println("ConsumeAndPeriodicFeed consume:", v1.maxTokens, v1.curTokens)
		t.Fail()
	}

	// Feed it with maxTokens
	time.Sleep(2000 * time.Millisecond)
	v1.PeriodicFeed()
	if v1.curTokens != v1.maxTokens {
		fmt.Println("ConsumeAndPeriodicFeed feed with maxTokens:", v1.maxTokens, v1.curTokens)
		t.Fail()
	}
}
