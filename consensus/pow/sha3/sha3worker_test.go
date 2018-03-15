/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package sha256

import (
	"math/big"
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
)

func Test_Worker(t *testing.T) {
	worker := getTestWorker()
	worker.StartAsync()

	worker.Wait()
	if !worker.Validate(worker.nonce) {
		t.Fail()
	}
}

func Test_WorkerStop(t *testing.T) {
	worker := getTestWorker()
	worker.StartAsync()

	time.Sleep(1 * time.Second)
	worker.Stop()

	worker.Wait()

	_, err := worker.GetResult()
	assert.Equal(t, err, ErrorStoppedByUser)

	if worker.Validate(worker.nonce) {
		t.Fail()
	}
}

func getTestWorker() *Worker {
	target := big.NewInt(1)
	target.Lsh(target, 256-20)

	worker := NewSha3Worker([]byte("6666"), target)

	return worker
}
