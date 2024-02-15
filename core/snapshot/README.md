# Snapshot Engine

The snapshot engine is responsible for collecting all in-memory state that exists across all other engines/services. The saved state can then be used to restore a node to a particular block height by propagating this state back into each engine. This can either be done using a local snapshot, where an existing node restarted, or via a network snapshot, where a new node joins and is gifted a snapshot from other nodes in the network.

Each engine which needs to save state will register themselves with the snapshot engine via a call to `engine.AddProviders()` and expose themselves through the below interface:

```
type StateProvider interface {
	Namespace() SnapshotNamespace
	Keys() []string
	GetState(key string) ([]byte, []StateProvider, error)
	LoadState(ctx context.Context, pl *Payload) ([]StateProvider, error)
	Stopped() bool
}
```

then at every snapshot-block, the snapshot engine will suck up all the state from each registered provider and save it to disk.

## Identifying state to snapshot

When we talk about an engine's state we mean "fields in an engine's data structure". More specifically, fields which hold data that exist across multiple blocks.

For example if we had an engine that looked like this:

```
type SomeEngine struct {

    cfg  Config
    log *logging.Logger

    // track orders, id -> order
    orders map[string]*types.Order

    // registered callbacks for whenever something happens
    cbs map[string]func() error
}
```

The important field that needs saving into a snapshot is `orders`. The fields `cfg` and `log` are only configuration fields and so are not relevant. For `cbs` the onus is on the subscriber to re-register their callback when they restore from a snapshot, so _this_ engine's snapshot need not worry.

### Gotcha 1: Cannot include validator-only state in a snapshot

Given that validator and non-validator nodes take snapshots and the hash of a snapshot is included in the commit-hash, if any state that is only present in _validator_ nodes is added to a snapshot, then all non-validator nodes will fall out of consensus.

An example of this is the Ethereum-event-forwarder which is only something validator nodes do. The state it contains is the Ethereum block-height that was last checked for events, but we cannot save this into a snapshot. We instead handle it by saving the Ethereum block-height of the last `ChainEvent` sent into core, which is a transaction all node types will see. This will not be the last Ethereum block-height checked, but it is good enough.

### Gotcha 2: Cannot include single-node state in a snapshot

This is similar to gotcha 1 but worth mentioning explicitly. Some engine's that send validator-commands back into the network will keep track of whether they should retry based on whether sending in that transaction was successful. This state is personal to an individual node and cannot be saved to the snapshot. It will cause a different snapshot hash than any other node and this node will fall out of consensus

For example the notary engine keeps track of whether the node needs to try sending in a node-signature again. The notary engine also keeps tracks of which nodes it has received signatures from. Therefore when restoring from a snapshot it's retry-state can be repopulate indirectly based on whether a node's own node-vote is in the set of received votes. If it is not there then it needs to retry.

### Gotcha 3: Remember to re-register callbacks between engines

The links between engines, whether that be the registration of callbacks or other things, is not state that can be saved into a snapshot but should be restored via re-registration when loading from a snapshot.

For example, a market subscribes to oracles for both termination and settlement. When a market is restored from a snapshot it must re-subscribe to those oracles again, but must do so depending on the market's restored state i.e a terminated market should only re-subscribe to the settlement oracle.

### Gotcha 4: Trying to use complex-logic to deduce whether a field needs adding to the snapshot

Its not worth it. Your assessment is probably wrong and will result in a horrible bug that presents itself 5 weeks later at the worst possible moment. Unless it is _plainly obvious_ that a field has a lifetime of less than a block, or it can _trivially_ be derived from another field, then just add it to the snapshot.

## Snapshot tests

Snapshot testing is in a good place. We have lots of layers that check for particular types of issues. The flavours of snapshot tests that exist today are:

- Unit-tests
- System-tests
- Snapshot soak tests
- Snapshot pipeline

### Unit-tests

Each engine that is a snapshot provider should have unit-tests that verify the roundtrip of saving and restoring the snapshot state.

Writing an effective unit-test for an engine's snapshot involves checking three things:

- completeness: all fields are saved and restored identically
- determinism: the same state serialises to the same bytes, always
- engine connections: subscriptions/callbacks to other engine's are re-subscribed

#### Testing completeness

The best way to check for completeness is to do the following:

- create an engine with some state
- call `.GetState()` to get the serialised state `b1`
- create a second engine and load in `b1`
- assert that all the fields in both engines are equal i.e `assert.Equal(t, eng1.GetOrders(), end2.GetOrders())`

#### Testing determinism

The best way to check for determinism is to do the following:

- create an engine with some state
- call `.GetState()` to get the serialised state `b1`
- create a second engine and load in `b1`
- call `.GetState()` on the second engine to get `b2`
- assert that `b1 == b2`

The main cause of non-determinism is when converting a `map -> slice`. Given that maps are unordered, the resultant slice must be sorted by the map's keys for it to serialised to the same byte string. This is why checking for completeness is not a sufficient test for determinism because the map will still restore exactly even though the snapshot is different. Equally checking that the snapshot is deterministic is not a sufficient test for completeness. For example if we had a field of type `time.Time{}` and saved its `t.Unix()` value in the snapshot, the snapshot would be reliably deterministic but the restored value will have lost the nanoseconds and not be identical to before.

#### Testing engine connections

The best way to check that subscriptions are restored is to do the following:

1. Create a first business and snapshot engines.
2. Add some state to the business engine.
3. Generate a local snapshot.
4. Add some more state to the business engine.
5. Get the state for all keys, and save the result in a map, on the business engine.
6. Close these first engines.
7. Create a second business and snapshot engines.
8. Restore the local snapshot.
9. Verify the hashes produced by the first and second snapshot.
10. Add more state exactly the same way as step 4.
11. Repeat step 5 but for the second business engine.
12. Compare the content of the step 5 and 11.

Below is a pseudo-code-ish example of what a snapshot unit-tests should look like:

```go
package business_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/stats"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngineSnapshot(t *testing.T) {
	ctx := vgtest.VegaContext("chainid", 100)

	log := logging.NewTestLogger()

	// This is important to use the same path so the engines pick up the same
	// database.
	vegaPath := paths.New(t.TempDir())

	now := time.Now()
	timeService := stubs.NewTimeStub()
	timeService.SetTime(now)

	statsData := stats.New(log, stats.NewDefaultConfig())

	// Create the engines
	businessEngine1 := NewBusinessEngine()

	// Do not use memory implementation! Use LevelDB to be as close as possible
	// to production.
	snapshotEngine1, err := snapshot.NewEngine(vegaPath, snapshot.DefaultConfig(), log, timeService, statsData.Blockchain)
	require.NoError(t, err)

	// This is to avoid double closing the engine.
	closeSnapshotEngine1 := vgtest.OnlyOnce(snapshotEngine1.Close)
	defer closeSnapshotEngine1()

	snapshotEngine1.AddProvider(businessEngine1)

	// No snapshot yet, does nothing.
	require.NoError(t, snapshotEngine1.Start(ctx))

	// This will help us to verify the engines are correctly wired when
	// restoring the state.
	populateState(t, businessEngine1)

	// Taking the first snapshot. Saved locally.
	// Call `SnapshotNow()`, and not `Snapshot()` as it's async and might create
	// a flickering test, or a data-race.
	hash1, err := snapshotEngine1.SnapshotNow(ctx)
	require.NoError(t, err)

	// This will help us to detect drift on between the business engines after
	// a restoration and an update.
	populateMoreState(t, businessEngine1)

	// Manually snapshotting the first business engine state, AFTER the second
	// update.
	state1 := map[string][]byte{}
	for _, key := range businessEngine1.Keys() {
		state, additionalProvider, err := businessEngine1.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state1[key] = state
	}

	// Closing the first engine now, so the second snapshot engine is the only one
	// connecting to the database. This is the closest to the production setup.
	closeSnapshotEngine1()

	// Create the engines
	businessEngine2 := NewBusinessEngine()
	snapshotEngine2, err := snapshot.NewEngine(vegaPath, snapshot.DefaultConfig(), log, timeService, statsData.Blockchain)
	require.NoError(t, err)
	defer snapshotEngine2.Close()

	snapshotEngine2.AddProviders(businessEngine2.Engine)

	// This triggers the state restoration from the local snapshot.
	require.NoError(t, snapshotEngine2.Start(ctx))

	// Comparing the hash after restoration, to ensure it produces the same result.
	hash2, _, _ := snapshotEngine2.Info()
	require.Equal(t, hash1, hash2)

	// Reproduce exactly the same state modification on the second business engine
	// as the first one, AFTER the snapshot.
	populateMoreState(t, businessEngine2)

	// Manually snapshotting the second business engine state, AFTER the second
	// update.
	state2 := map[string][]byte{}
	for _, key := range businessEngine2.Keys() {
		state, additionalProvider, err := businessEngine2.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state2[key] = state
	}

	// Attempt to detect any drift in the data.
	// If the data don't match, check for missing or non-deterministic data in
	// the snapshot.
	for key := range state1 {
		assert.Equalf(t, state1[key], state2[key], "Key %q does not have the same data", key)
	}
}

```

### System tests

System-tests exist that directly flex snapshots in known troublesome situations, and also check more functional aspects of snapshots (they are produced in line with the network parameter, we only save as many as set in the config file etc etc). These exist in the test file `snapshot_test.py` in the system-test repo.

There are also tests that do not _directly_ test snapshot behaviour but where snapshots are used by that feature, for example validators-joining-and-leave and protocol-upgrade tests. These tests exist across almost all of the system-tests marked as `network_infra`.

#### How to debug a failure

For any run of a system-test the block-data and vega home directories are saved as artefacts. They can be downloaded, used to replay the chain locally, and to then perform the same snapshot restored. The block of the failing snapshot can be found in the logs of the node that failed to restart.

### Snapshot soak tests

The "snapshot soak tests" are run at the end of every overnight full system-test run. They take the resultant chain data generated by running the full test suite, replays the chain, and then attempts to restore from every snapshot that was taken during the lifetime of the chain. The benefit of these tests is that they check snapshots that are created during obscure transient states which are harder to dream up when writing snapshot system-tests or unit-tests.

It also means that our effective coverage of snapshots mirror the system-test AC coverage, and as new system-tests for features are written we automatically get testing that the snapshots for those features also work.

#### How to debug a failure

Reproducing a failed soak-test locally is very easy as you can trivially use the same script as the CI. The steps are:

- Download the `testnet` folder of artefacts from the system-test run that produced the bad snapshot
- Clone the `system-tests` repo and find the script `tests/soak-test/run.py`
- Run the script to first replay the chain: `python3 run.py --tm-home=tendermint/node2 --vega-home=vega/node2 --vega-binary=../vega --replay`
- It will write logs files from the node to `node-0.log` and `err-node-0.log`
- Restart the node from the problem snapshot `python3 run.py --tm-home=tendermint/node2 --vega-home=vega/node2 --vega-binary=../vega --block BLOCK_NUM`
- It will write log files from the node to `node-BLOCK_NUM.log` and `err-node-BLOCK_NUM.log`
- Compare the two logs to see where state has diverged

### Snapshot pipelines

A reoccuring Jenkins pipeline exists that will repeatedly join a network using statesync snapshots. The pipeline runs every 10mins on all of our internal networks (devnet1, stagnet1, testnet) as well as mainnet. There is a slack channel `#snapshot-notify` the show the results.

The pipeline exists to verify that snapshots work in a more realistic environment where the volume of state is more representative of what we would expect on a real Vega network.

#### How to debug a failure

The snapshot pipeline jobs will store as an artefact the block-data and the snapshot it loaded from. This allows you to replay the partial chain the in same way locally and reproduce any failure. By finding the last _successful_ snapshot pipeline job, those artefacts can be used to replay the partial chain from a working snapshot allowing comparison between logs to find where state started to diverge.

### Event-diff soak tests

It checks that the events sent out by core are always sent out in the same order. As the system tests are running `node2` writes out all the events it sends to a data node to a file. At the end of the test run the chain is replayed and another file containing all events is produce. These two files are then diffed. If it fails on Jenkins then you will see output that looks like the following:

```
[2023-10-09T15:24:45.689Z] === Starting event-file diff
[2023-10-09T15:24:45.689Z] Differences found between: /jenkins/workspace/common/system-tests-wrapper/networkdata/testnet/vega/node2/eventlog.evt /jenkins/workspace/common/system-tests-wrapper/networkdata/testnet/vega/node2/eventlog-replay.evt
```

#### How to debug a failure

The two event files that were diffed will be saved as artefacts in Jenkins, and the first step is to download them locally. From there they are be parsed into human-readable JSON by using the following vega tool,a nd then diffed:

```
vega tools events --out=original.out --events=eventlog.evt

vega tools events --out=replay.out --events=eventlog-replay.evt

diff original.out replay.out
```

Note that for events created during a full system-test run, both the parsing and diff can take some time.

The diff can then be used to hunt down which block produces different events, and which event type it is. For example if the diff flagged up and event like below:

```
{
   "id": "4615-75",
   "block": "953A2BC530B192B78CA5D4228C377BF3C66FEA65F0C4AF93B0DDBE7AFDE036A7",
   "type": 11,
   "vote": {
      "party_id": "7c2860d661607c3e51df31f7fae478acceb6ad0f45ef0d044b74d37cf7f78ebc",
      "value": 2,
      "proposal_id": "2372a4901660ace8d4e5b9e318754abdbf959454610878c13fd8f73317ebacbd",
      "timestamp": "1696858255513756974",
      "total_governance_token_balance": "9749958450000000000",
      "total_governance_token_weight": "1",
      "total_equity_like_share_weight": "0"
   },
   "version": 1,
   "chain_id": "testnet-001",
   "tx_hash": "953A2BC530B192B78CA5D4228C377BF3C66FEA65F0C4AF93B0DDBE7AFDE036A7"
}
```

we know that it was emitted in block `4615` and was for a `vote` event. From here we can look through all calls to `events.NewVoteEvent()` in core and look for places where we may be iterating over a map, or sorting that may be insufficient, and will cause a different in event order.
