package core

import "errors"

var (
	// errInconsistentSubjects is returned when received subject is different from
	// current subject.
	errInconsistentSubjects = errors.New("not consistent subjects")
	// errNotProposer is returned when received message is supposed to be from
	// proposer.
	errNotProposer = errors.New("message does not from proposer")
	// errMsgIgnored is returned when a message was ignored.
	errMsgIgnored = errors.New("message ignored")
	// errMsgFromFuture is returned when current view is earlier than the
	// view of the received message.
	errMsgFromFuture = errors.New("future message")
	// errOldMsg is returned when the received message's view is earlier
	// than current view.
	errOldMsg = errors.New("old message")
	// errInvalidMsg is returned when the message is malformed.
	errInvalidMsg = errors.New("invalid message")
	// errDecodePreprepare is returned when the PRE-PREPARE message is malformed.
	errDecodePreprepare = errors.New("failed to decode PRE-PREPARE message")
	// errDecodePrepare is returned when the PREPARE message is malformed.
	errDecodePrepare = errors.New("failed to decode PREPARE message")
	// errDecodeCommit is returned when the COMMIT message is malformed.
	errDecodeCommit = errors.New("failed to decode COMMIT message")
	// errDecodeMessageSet is returned when the message set is malformed.
	errDecodeMessageSet = errors.New("failed to decode messageset")

	// ErrAddressUnauthorized is returned when given address cannot be found in
	// current validator set.
	ErrAddressUnauthorized = errors.New("unauthorized address")
	// ErrEngineStopped is returned if the engine is stopped
	ErrEngineStopped = errors.New("stopped engine")
	// ErrEngineStarted is returned if the engine is already started
	ErrEngineStarted = errors.New("started engine")
)
