package pow

// Worker is an PoW engine.
type Worker interface {
    // Returns the current mining result of a PoW consensus engine.
    Produce() string

    // Verify whether the mining result meet the requirement.
    Validate() bool
}
