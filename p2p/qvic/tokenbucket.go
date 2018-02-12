/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package qvic

import (
	"sync"

	"github.com/aristanetworks/goarista/monotime"
)

// TokenBucket for bucket limit rate
type TokenBucket struct {
	bytesPerSecond int64 // bandwidth
	maxTokens      int64 // size of bucket
	curTokens      int64
	tokensPerMS    int64  // tokens produced every milliseconds
	reservedTokens int64  // reserved tokens
	preTick        uint64 // last tick producing tokens
	mutex          sync.Mutex
}

// Init initializes TokenBucket with bytesPerSecond and factor
func (t *TokenBucket) Init(bytesPerSecond int64) {
	t.initCommon(bytesPerSecond)
	t.curTokens = t.maxTokens
	t.preTick = monotime.Now() / 1000
}

// AdjustBW adjusts bandwidth anytime
func (t *TokenBucket) AdjustBW(bytesPerSecond int64) {
	t.initCommon(bytesPerSecond)
	if t.curTokens > t.maxTokens {
		t.curTokens = t.maxTokens
	}
}

// PeriodicFeed called periodically, for example 30ms
func (t *TokenBucket) PeriodicFeed() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	cur := monotime.Now() / 1000
	i := t.curTokens + int64(cur-t.preTick)*t.tokensPerMS
	if i > t.maxTokens {
		t.curTokens = t.maxTokens
	} else {
		t.curTokens = i
	}
	t.preTick = cur
}

// GetCurTokens gets valid tokens
func (t *TokenBucket) GetCurTokens() int64 {
	return t.curTokens
}

// GetCurLooseTokens gets valid tokens, minus reserved
func (t *TokenBucket) GetCurLooseTokens() int64 {
	return t.curTokens - t.reservedTokens
}

// Consume consumes some buckets
func (t *TokenBucket) Consume(tokens int64) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.curTokens -= tokens
}

func (t *TokenBucket) initCommon(bytesPerSecond int64) {
	t.bytesPerSecond = bytesPerSecond
	t.maxTokens = bytesPerSecond
	t.tokensPerMS = bytesPerSecond / 1000
	t.reservedTokens = bytesPerSecond / 10
}
