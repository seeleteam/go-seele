/*
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package bft

type ProposerPolicy uint64

const (
	RoundRobin ProposerPolicy = iota
	Sticky                    // with sticky property
)

type BFTConfig struct {
	RequestTimeout uint64         `toml:",omitempty"` // The timeout for each Istanbul round in milliseconds.
	BlockPeriod    uint64         `toml:",omitempty"` // Default minimum difference between two consecutive block's timestamps in second
	ProposerPolicy ProposerPolicy `toml:",omitempty"` // The policy for proposer selection
	Epoch          uint64         `toml:",omitempty"` // The number of blocks after which to checkpoint and reset the pending votes
}

var DefaultConfig = &BFTConfig{
	RequestTimeout: 10000,
	BlockPeriod:    1,
	ProposerPolicy: RoundRobin, // we use RoundRobin policy (others are Random/RoundRobin/LesatBusy/StickySession/cookies/ByReqeust)
	Epoch:          30000,
}
