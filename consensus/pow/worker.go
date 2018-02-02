package pow

// Worker is an PoW engine.
type Worker interface {
    // Returns the current mining result of a PoW consensus engine.
    Producer() string

    // Verify whether the mining result is meet the requirement.
    Validator() bool
}
