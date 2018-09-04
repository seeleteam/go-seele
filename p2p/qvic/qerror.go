/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package qvic

// QTimeoutErr implements net.Error interface, in order to provide unified interface as tcp does.
type QTimeoutErr struct {
	s string
}

func NewQTimeoutError(s string) *QTimeoutErr {
	e := &QTimeoutErr{
		s: s,
	}
	return e
}

func (e *QTimeoutErr) Error() string {
	return e.s
}

func (e *QTimeoutErr) Timeout() bool {
	return true
}

func (e *QTimeoutErr) Temporary() bool {
	return true
}
