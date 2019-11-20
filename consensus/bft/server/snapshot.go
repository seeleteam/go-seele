/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package server

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/seeleteam/go-seele/common"
	"github.com/seeleteam/go-seele/consensus/bft"
	"github.com/seeleteam/go-seele/consensus/bft/verifier"
	"github.com/seeleteam/go-seele/core/types"
	"github.com/seeleteam/go-seele/database"
)

const (
	dbKeySnapshotPrefix = "bft-snapshot"
)

// Vote represents a single vote that an authorized verifier made to modify the
// list of authorizations.
type Vote struct {
	Verifier  common.Address `json:"verifier"`  // Authorized verifier that cast this vote
	Block     uint64         `json:"block"`     // Block number the vote was cast in (expire old votes)
	Address   common.Address `json:"address"`   // Account being voted on to change its authorization
	Authorize bool           `json:"authorize"` // Whether to authorize or deauthorize the voted account
}

// Tally is a simple vote tally to keep the current score of votes. Votes that
// go against the proposal aren't counted since it's equivalent to not voting.
type Tally struct {
	Authorize bool `json:"authorize"` // Whether the vote it about authorizing or kicking someone
	Votes     int  `json:"votes"`     // Height of votes until now wanting to pass the proposal
}

// Snapshot is the state of the authorization voting at a given point in time.
type Snapshot struct {
	Epoch uint64 // The number of blocks after which to checkpoint and reset the pending votes

	Height uint64                   // Block height where the snapshot was created
	Hash   common.Hash              // Block hash where the snapshot was created
	Votes  []*Vote                  // List of votes cast in chronological order
	Tally  map[common.Address]Tally // Current vote tally to avoid recalculating
	VerSet bft.VerifierSet          // Set of authorized verifiers at this moment
}

// newSnapshot create a new snapshot with the specified startup parameters. This
// method does not initialize the set of recent verifiers, so only ever use if for
// the genesis block.
func newSnapshot(epoch uint64, number uint64, hash common.Hash, verSet bft.VerifierSet) *Snapshot {
	snap := &Snapshot{
		Epoch:  epoch,
		Height: number,
		Hash:   hash,
		VerSet: verSet,
		Tally:  make(map[common.Address]Tally),
	}
	return snap
}

// loadSnapshot loads an existing snapshot from the database.
func loadSnapshot(epoch uint64, db database.Database, hash common.Hash) (*Snapshot, error) {
	blob, err := db.Get(append([]byte(dbKeySnapshotPrefix), hash[:]...))
	if err != nil {
		return nil, err
	}
	snap := new(Snapshot)
	if err := json.Unmarshal(blob, snap); err != nil {
		return nil, err
	}
	snap.Epoch = epoch

	return snap, nil
}

// store inserts the snapshot into the database.
func (s *Snapshot) store(db database.Database) error {
	blob, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return db.Put(append([]byte(dbKeySnapshotPrefix), s.Hash[:]...), blob)
}

// copy creates a deep copy of the snapshot, though not the individual votes.
func (s *Snapshot) copy() *Snapshot {
	cpy := &Snapshot{
		Epoch:  s.Epoch,
		Height: s.Height,
		Hash:   s.Hash,
		VerSet: s.VerSet.Copy(),
		Votes:  make([]*Vote, len(s.Votes)),
		Tally:  make(map[common.Address]Tally),
	}

	for address, tally := range s.Tally {
		cpy.Tally[address] = tally
	}
	copy(cpy.Votes, s.Votes)

	return cpy
}

// checkVote return whether it's a valid vote
func (s *Snapshot) checkVote(address common.Address, authorize bool) bool {
	_, verifier := s.VerSet.GetVerByAddress(address)
	return (verifier != nil && !authorize) || (verifier == nil && authorize)
}

// cast adds a new vote into the tally.
func (s *Snapshot) cast(address common.Address, authorize bool) bool {
	// Ensure the vote is meaningful
	if !s.checkVote(address, authorize) {
		return false
	}
	// Cast the vote into an existing or new tally
	if old, ok := s.Tally[address]; ok {
		old.Votes++
		s.Tally[address] = old
	} else {
		s.Tally[address] = Tally{Authorize: authorize, Votes: 1}
	}
	return true
}

// uncast removes a previously cast vote from the tally.
func (s *Snapshot) uncast(address common.Address, authorize bool) bool {
	// If there's no tally, it's a dangling vote, just drop
	tally, ok := s.Tally[address]
	if !ok {
		return false
	}
	// Ensure we only revert counted votes
	if tally.Authorize != authorize {
		return false
	}
	// Otherwise revert the vote
	if tally.Votes > 1 {
		tally.Votes--
		s.Tally[address] = tally
	} else {
		delete(s.Tally, address)
	}
	return true
}

// apply creates a new authorization snapshot by applying the given headers to
// the original one.
func (s *Snapshot) apply(headers []*types.BlockHeader) (*Snapshot, error) {
	// Allow passing in no headers for cleaner code
	if len(headers) == 0 {
		return s, nil
	}
	// Sanity check that the headers can be applied
	for i := 0; i < len(headers)-1; i++ {
		if headers[i+1].Height != headers[i].Height+1 {
			return nil, errVotingChainInvalid
		}
	}
	if headers[0].Height != s.Height+1 {
		return nil, errVotingChainInvalid
	}
	// Iterate through the headers and create a new snapshot
	snap := s.copy()

	for _, header := range headers {
		// Remove any votes on checkpoint blocks
		number := header.Height
		if number%s.Epoch == 0 {
			snap.Votes = nil
			snap.Tally = make(map[common.Address]Tally)
		}
		// Resolve the authorization key and check against verifiers
		verifier, err := extractAccount(header)
		if err != nil {
			return nil, err
		}
		if _, v := snap.VerSet.GetVerByAddress(verifier); v == nil {
			return nil, errUnauthorized
		}

		// Header authorized, discard any previous votes from the verifier
		for i, vote := range snap.Votes {
			if vote.Verifier == verifier && vote.Address == header.Creator {
				// Uncast the vote from the cached tally
				snap.uncast(vote.Address, vote.Authorize)

				// Uncast the vote from the chronological list
				snap.Votes = append(snap.Votes[:i], snap.Votes[i+1:]...)
				break // only one vote allowed
			}
		}
		// Tally up the new vote from the verifier
		var authorize bool
		switch {
		case bytes.Compare(header.Witness[:], nonceAuthVote) == 0:
			authorize = true
		case bytes.Compare(header.Witness[:], nonceDropVote) == 0:
			authorize = false
		default:
			return nil, errVoteInvalid
		}
		if snap.cast(header.Creator, authorize) {
			snap.Votes = append(snap.Votes, &Vote{
				Verifier:  verifier,
				Block:     number,
				Address:   header.Creator,
				Authorize: authorize,
			})
		}
		// If the vote passed, update the list of verifiers
		if tally := snap.Tally[header.Creator]; tally.Votes > snap.VerSet.Size()/2 {
			if tally.Authorize {
				snap.VerSet.AddVerifier(header.Creator)
			} else {
				snap.VerSet.RemoveVerifier(header.Creator)

				// Discard any previous votes the deauthorized verifier cast
				for i := 0; i < len(snap.Votes); i++ {
					if snap.Votes[i].Verifier == header.Creator {
						// Uncast the vote from the cached tally
						snap.uncast(snap.Votes[i].Address, snap.Votes[i].Authorize)

						// Uncast the vote from the chronological list
						snap.Votes = append(snap.Votes[:i], snap.Votes[i+1:]...)

						i--
					}
				}
			}
			// Discard any previous votes around the just changed account
			for i := 0; i < len(snap.Votes); i++ {
				if snap.Votes[i].Address == header.Creator {
					snap.Votes = append(snap.Votes[:i], snap.Votes[i+1:]...)
					i--
				}
			}
			delete(snap.Tally, header.Creator)
		}
	}
	snap.Height += uint64(len(headers))
	snap.Hash = headers[len(headers)-1].Hash()

	return snap, nil
}

// verifiers retrieves the list of authorized verifiers in ascending order.
func (s *Snapshot) verifiers() []common.Address {
	fmt.Printf("snapshot verset %s", s.VerSet.GetByIndex(0))
	verifiers := make([]common.Address, 0, s.VerSet.Size())
	for _, verifier := range s.VerSet.List() {
		verifiers = append(verifiers, verifier.Address())
	}
	for i := 0; i < len(verifiers); i++ {
		for j := i + 1; j < len(verifiers); j++ {
			if bytes.Compare(verifiers[i][:], verifiers[j][:]) > 0 {
				verifiers[i], verifiers[j] = verifiers[j], verifiers[i]
			}
		}
	}
	return verifiers
}

type snapshotJSON struct {
	Epoch  uint64                   `json:"epoch"`
	Number uint64                   `json:"number"`
	Hash   common.Hash              `json:"hash"`
	Votes  []*Vote                  `json:"votes"`
	Tally  map[common.Address]Tally `json:"tally"`

	// for verifier set
	Verifiers []common.Address   `json:"verifiers"`
	Policy    bft.ProposerPolicy `json:"policy"`
}

func (s *Snapshot) toJSONStruct() *snapshotJSON {
	return &snapshotJSON{
		Epoch:     s.Epoch,
		Number:    s.Height,
		Hash:      s.Hash,
		Votes:     s.Votes,
		Tally:     s.Tally,
		Verifiers: s.verifiers(),
		Policy:    s.VerSet.Policy(),
	}
}

// Unmarshal from a json byte array
func (s *Snapshot) UnmarshalJSON(b []byte) error {
	var j snapshotJSON
	if err := json.Unmarshal(b, &j); err != nil {
		return err
	}

	s.Epoch = j.Epoch
	s.Height = j.Number
	s.Hash = j.Hash
	s.Votes = j.Votes
	s.Tally = j.Tally
	s.VerSet = verifier.NewVerifierSet(j.Verifiers, j.Policy)
	return nil
}

// Marshal to a json byte array
func (s *Snapshot) MarshalJSON() ([]byte, error) {
	j := s.toJSONStruct()
	return json.Marshal(j)
}
