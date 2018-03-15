/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package pow

// Worker is an PoW engine.
type Worker interface {

	// GetResult if got error, will return the error info
	GetResult() (string, error)

	// Wait return when worker is complete successful or stopped
	Wait()

	// StartAsync start to find nonce async
	StartAsync()

	// Validate Verify nonce to find the target value that meet the requirement.
	Validate(nonce string) bool

	// Stop calculation
	Stop()
}
