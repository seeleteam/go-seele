/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package consensus

import "errors"

var (
	// ErrBlockInvalidHeight is returned when block nonce is invalid
	ErrBlockNonceInvalid = errors.New("invalid block nonce")

	// ErrBlockInvalidHeight is returned when inserting a new header with invalid block height.
	ErrBlockInvalidHeight = errors.New("invalid block height")

	// ErrBlockCreateTimeOld is returned when block create time is previous of parent block time
	ErrBlockCreateTimeOld = errors.New("block time must be later than parent block time")

	// ErrBlockInvalidParentHash is returned when inserting a new header with invalid parent block hash.
	ErrBlockInvalidParentHash = errors.New("invalid parent block hash")

	// ErrBlockDifficultInvalid is returned when block difficult is invalid
	ErrBlockDifficultInvalid = errors.New("block difficult is invalid")
)
