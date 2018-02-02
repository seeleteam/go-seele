package sha256

import (
    "math/big"
    "testing"
)

func Test_Worker(t *testing.T) {
    var worker *Worker = new(Worker)
    worker.Data = []byte("6666")
    worker.Nonce = ""
    worker.Target = big.NewInt(10)
    worker.Target.Lsh(worker.Target, uint(256 - 20))

    worker.Nonce = worker.Producer()
    if !worker.Validator() {
        t.Error("Error")
    }
}
