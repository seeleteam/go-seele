package server

const (
	checkpointInterval = 1024 // Height of blocks after which to save the vote snapshot to the database
	inmemorySnapshots  = 128  // Height of recent vote snapshots to keep in memory
	inmemoryPeers      = 40
	inmemoryMessages   = 1024
)
