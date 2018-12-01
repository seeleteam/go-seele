/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package log

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

var globalAuditorID uint64

// Auditor is used for auditing step by step via log.
type Auditor struct {
	id        uint64
	log       *SeeleLog
	method    string
	enterTime time.Time // timestamp for enter.
	lastTime  time.Time // timestamp for last audit.
}

// NewAuditor returns a new auditor instance with specified log and an optional last time.
func NewAuditor(log *SeeleLog) *Auditor {
	return &Auditor{
		id:       atomic.AddUint64(&globalAuditorID, 1),
		log:      log,
		lastTime: time.Now(),
	}
}

// Audit adds log for the specified parameterized message.
func (a *Auditor) Audit(format string, args ...interface{}) {
	if a.log.GetLevel() > logrus.DebugLevel {
		return
	}

	now, msg := time.Now(), fmt.Sprintf(format, args...)
	a.log.Debug("[Audit_%v] %v (elapsed: %v)", a.id, msg, now.Sub(a.lastTime))
	a.lastTime = now
}

// AuditEnter adds log for method enter.
func (a *Auditor) AuditEnter(method string) {
	if a.log.GetLevel() > logrus.DebugLevel {
		return
	}

	a.method, a.enterTime = method, time.Now()
	a.log.Debug("[Audit_%v] enter %v", a.id, method)
}

// AuditLeave adds log for method leave.
func (a *Auditor) AuditLeave() {
	if a.log.GetLevel() > logrus.DebugLevel {
		return
	}

	a.log.Debug("[Audit_%v] leave %v (elapsed: %v)", a.id, a.method, time.Since(a.enterTime))
}
