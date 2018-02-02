package sha256

import (
    "math/big"
    "testing"
)

func Test_SHA256Worker(t *testing.T) {
    var sha256worker *SHA256Worker = new(SHA256Worker)
    sha256worker.Data = []byte("6666")
    sha256worker.Nonce = ""
    sha256worker.Target = big.NewInt(10)
    sha256worker.Target.Lsh(sha256worker.Target, uint(256 - 20))

    sha256worker.Nonce = sha256worker.Producer()
    if !sha256worker.Validator() {
        t.Error("Error")
    }
}
