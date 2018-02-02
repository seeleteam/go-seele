package consensus

import (
    "bytes"
    "crypto/sha256"
    "math"
    "math/big"
    "strconv"
)

// data isathe pointer to the target block data,
// target is the goal, which means we need to mine a result and its hash must be less than the target.
type SHA256Worker struct {
    Data   []byte
    Nonce  string
    Target *big.Int
}

// Constructs the data that need to be verfied.
func (worker *SHA256Worker) PrepareData(nonce string) []byte {
    data := bytes.Join(
        [][]byte{
            worker.Data,
            []byte(nonce),
        },
        []byte{},
    )

    return data
}

// Loop nonce to find the target value that meet the requirement.
func (worker *SHA256Worker) Producer() string {
    var nonce int64 = 0
    var hash [32]byte
    var hashInt big.Int

    for nonce < math.MaxInt64 {
        data := worker.PrepareData(strconv.FormatInt(nonce, 10))
        hash = sha256.Sum256(data)
        hashInt.SetBytes(hash[:])

        if hashInt.Cmp(worker.Target) == -1 {
            break
        } else {
            nonce++
        }
    }

    return strconv.FormatInt(nonce, 10)
}

// Verify nonce to find the target value that meet the requirement.
func (worker *SHA256Worker) Validator() bool {
    var hashInt big.Int

    data := worker.PrepareData(worker.Nonce)
    hash := sha256.Sum256(data)
    hashInt.SetBytes(hash[:])

    isValid := hashInt.Cmp(worker.Target) == -1

    return isValid
}

