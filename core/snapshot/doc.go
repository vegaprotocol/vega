package snapshot

// Package snapshot implements the snapshot engine responsible of taking a
// snapshot of the node state, that will allow a node to restore the state later
// on without replaying the entire chain.
//
// # Taking a snapshot
//
// The engine gathers the state from the state providers that subscribed to it.
// A state provider can be any component that needs to have its state snapshotted
// and restored to work.
//
// A snapshot is taken at defined interval counted in "block". For example,
// every 10 blocks. It happens when Tendermint calls `Commit()` on the node.
//
// The snapshot is taken asynchronously on the providers.
//
// # Restoration modes
//
// There are 2 ways to load a snapshot:
//   - Loading from a local snapshot, stored on disk,
//   - Loading from the network via Tendermint state-sync.
//
// Once a snapshot is loaded by it engine, the engine distributes the deserialized
// state to the state providers. Once the state is restored, the engine will
// reject any new attempt to restore another state. The node will have to be
// shutdown and reset to apply another state.
//
// ## Restore from local snapshots
//
// Local snapshots are stored locally in a LevelDB database. Snapshots metadata
// are stored in a separate database to go easy on machine resources when looking
// for high-level information, that avoid loading the entire snapshot in memory.
//
// If there is any snapshot stored locally, the snapshot engine will load from
// the last one (or the one matching the configured block height), regardless of
// whether Tendermint's state-sync is enabled or not.
//
// To prevent this loading from this mode, the local snapshots must be entirely
// removed. This can be achieved using the command-line `vega unsafe-reset-all`.
//
// ## Restore from state-sync
//
// Restoring from this mode requires that no snapshot is stored locally.
//
// If there is no local snapshot, the snapshot engine will wait for Tendermint to
// offer a snapshot, through the method (*Engine).ReceiveSnapshot(). The snapshot
// will have to match the expectation of the engine, and if it does, it listens
// for incoming snapshot chunks distributed by network peers, through the
// method (*Engine).ReceiveSnapshotChunk().
//
// If Tendermint fails to retrieve all the chunks, it will automatically abort the
// on-going state-sync, and offer another snapshot to the engine, and go all over
// the chunk distribution process, again. The engine reacts to that change accordingly
// by resetting the process.
//
// When the snapshot is completed, the engine restores the state, and save the
// received snapshot locally.
//
// # Sharing snapshots with peers
//
// To share snapshots with peers, the node must have a snapshot saved locally.
// Tendermint asks which snapshots the node currently have via the method
// (*Engine).ListSnapshots(). If the snapshot Tendermint is looking for is present,
// it then asks for chunks of that snapshot. The node have to load the snapshot in
// memory, and look for the asked chunk.
