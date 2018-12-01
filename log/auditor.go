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
func NewAuditor(log *SeeleLog, lastTime ...time.Time) *Auditor {
	auditor := &Auditor{
		id:  atomic.AddUint64(&globalAuditorID, 1),
		log: log,
	}

	if len(lastTime) == 0 {
		auditor.lastTime = time.Now()
	} else {
		auditor.lastTime = lastTime[0]
	}

	return auditor
}

// Audit adds log for the specified parameterized message.
func (a *Auditor) Audit(format string, args ...interface{}) {
	if a.log.GetLevel() > logrus.DebugLevel {
		return
	}

	now := time.Now()
	a.log.Debug("[Audit] | [%v] | %v (elapsed: %v)", a.id, fmt.Sprintf(format, args...), now.Sub(a.lastTime))
	a.lastTime = now
}

// AuditEnter adds log for method enter.
func (a *Auditor) AuditEnter(method string) {
	if a.log.GetLevel() > logrus.DebugLevel {
		return
	}

	a.method = method
	a.enterTime = time.Now()
	a.log.Debug("[Audit] | [%v] | enter %v (elapsed: %v)", a.id, method, a.enterTime.Sub(a.lastTime))
}

// AuditLeave adds log for method leave.
func (a *Auditor) AuditLeave() {
	if a.log.GetLevel() > logrus.DebugLevel {
		return
	}

	a.log.Debug("[Audit] | [%v] | leave %v (elapsed: %v)", a.id, a.method, time.Since(a.enterTime))
}
