/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package log

import (
	"fmt"
	"time"
)

// Auditor is used for auditing step by step via log.
type Auditor struct {
	log       *SeeleLog
	method    string
	enterTime time.Time // timestamp for enter.
	lastTime  time.Time // timestamp for last audit.
}

// NewAuditor returns a new auditor instance with specified log and an optional last time.
func NewAuditor(log *SeeleLog, lastTime ...time.Time) *Auditor {
	auditor := &Auditor{
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
	now := time.Now()
	a.log.Debug("[Audit] %v (elapsed: %v)", fmt.Sprintf(format, args...), now.Sub(a.lastTime))
	a.lastTime = now
}

// AuditEnter adds log for method enter.
func (a *Auditor) AuditEnter(method string) {
	a.method = method
	a.enterTime = time.Now()
	a.log.Debug("[Audit] enter %v (elapsed: %v)", method, a.enterTime.Sub(a.lastTime))
}

// AuditLeave adds log for method leave.
func (a *Auditor) AuditLeave() {
	a.log.Debug("[Audit] leave %v (elapsed: %v)", a.method, time.Since(a.enterTime))
}
