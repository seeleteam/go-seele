package bft

import "errors"

var (
	// ErrAddressUnauthorized is returned when given address cannot be found in
	// current validator set.
	ErrAddressUnauthorized = errors.New("unauthorized address")
	// ErrEngineStopped is returned if the engine is stopped
	ErrEngineStopped = errors.New("stopped engine")
	// ErrEngineStarted is returned if the engine is already started
	ErrEngineStarted = errors.New("started engine")
)
