/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package sha256

import (
	"bytes"
	"errors"
	"math"
	"math/big"
	"strconv"
	"sync"

	"github.com/seeleteam/go-seele/crypto"
)

var (
	ErrorStoppedByUser = errors.New("worker is stopped by user")
)

// Worker data is a  pointer to the target block data,
// target is the goal, which means we need to mine a result and
// its hash must be less than the target.
type Worker struct {
	data   []byte
	nonce  string
	target *big.Int

	isStop bool
	wg     sync.WaitGroup
}

func NewSha3Worker(data []byte, target *big.Int) *Worker {
	return &Worker{
		data:   data,
		nonce:  "",
		target: target,
		isStop: false,
	}
}

// prepareData Constructs the data that need to be verfied.
func (w *Worker) prepareData(nonce string) []byte {
	data := bytes.Join(
		[][]byte{
			w.data,
			[]byte(nonce),
		},
		[]byte{},
	)

	return data
}

// Start Loop nonce to find the target value that meet the requirement.
func (w *Worker) start() {
	defer w.wg.Done()

	var nonce int64
	var hashInt big.Int
	for nonce < math.MaxInt64 {
		if w.isStop {
			break
		}

		data := w.prepareData(strconv.FormatInt(nonce, 10))
		hash := crypto.Keccak256Hash(data)
		hashInt.SetBytes(hash[:])

		if hashInt.Cmp(w.target) == -1 {
			break
		} else {
			nonce++
		}
	}

	w.nonce = strconv.FormatInt(nonce, 10)
}

// GetResult if got error, will return the error info
func (w *Worker) GetResult() (string, error) {
	if w.isStop {
		return "", ErrorStoppedByUser
	}

	return w.nonce, nil
}

// Wait return when worker is complete successful or stopped
func (w *Worker) Wait() {
	w.wg.Wait()
}

// StartAsync start to find nonce async
func (w *Worker) StartAsync() {
	w.wg.Add(1)
	go w.start()
}

// Validate Verify nonce to find the target value that meet the requirement.
func (w *Worker) Validate(nonce string) bool {
	var hashInt big.Int

	data := w.prepareData(nonce)
	hash := crypto.Keccak256Hash(data)
	hashInt.SetBytes(hash[:])

	isValid := hashInt.Cmp(w.target) == -1

	return isValid
}

func (w *Worker) Stop() {
	w.isStop = true
}
