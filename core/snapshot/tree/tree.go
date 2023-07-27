package tree

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
	"github.com/cosmos/iavl"
	tmtypes "github.com/tendermint/tendermint/abci/types"
)

// Tree is a high-level structure that abstract the management of the AVL away
// from the snapshot engine. It ensure the snapshot and metadata databases are
// in-sync, and old snapshots are removed when exceeding the maximum number to
// keep configured.
// When created, it will automatically remove old local snapshots and load the
// ones left.
// When the tree is no longer used, the resources it holds must be released by
// calling Release().
type Tree struct {
	log *logging.Logger

	blockHeightToStartFrom     uint64
	maxNumberOfSnapshotsToKeep uint64

	metadataDB MetadataDatabase
	snapshotDB SnapshotsDatabase

	innerTree *iavl.MutableTree
}

func (t *Tree) HasSnapshotsLoaded() bool {
	return !t.metadataDB.IsEmpty()
}

func (t *Tree) Hash() []byte {
	// Not returning the error as it is fairly unlikely to fail, and that makes
	// theis API simpler to deal with.
	hash, err := t.innerTree.Hash()
	if err != nil {
		t.log.Error("Could not computing the tree hash", logging.Error(err))
	}

	// When no tree has been saved, the underlying root from which the hash is
	// computed is nil. When nil, a "default" hash is returned. In our case,
	// if there is no saved tree, we want a nil hash to avoid misinterpretation.
	if bytes.Equal(hash, sha256.New().Sum(nil)) {
		return nil
	}

	return hash
}

func (t *Tree) WorkingHash() []byte {
	hash, err := t.innerTree.WorkingHash()
	if err != nil {
		t.log.Error("Could not computing the working tree hash", logging.Error(err))
	}

	return hash
}

func (t *Tree) RemoveKey(key []byte) bool {
	if ok, _ := t.innerTree.Has(key); ok {
		_, removed, _ := t.innerTree.Remove(key)
		return removed
	}
	return false
}

func (t *Tree) AddState(key []byte, state []byte) {
	_, _ = t.innerTree.Set(key, state)
}

func (t *Tree) AsPayloads() ([]*types.Payload, error) {
	lastSnapshotTree, err := t.innerTree.GetImmutable(t.innerTree.Version())
	if err != nil {
		return nil, fmt.Errorf("could not generate the immutable AVL tree: %w", err)
	}

	exporter, err := lastSnapshotTree.Export()
	if err != nil {
		return nil, fmt.Errorf("could not export the AVL tree: %w", err)
	}
	defer exporter.Close()

	payloads := []*types.Payload{}

	exportedNode, err := exporter.Next()
	for err == nil {
		// If there is no value, it means the node is an intermediary node and
		// not a leaf. Only leaves hold the data we are looking for.
		if exportedNode.Value == nil {
			exportedNode, err = exporter.Next()
			continue
		}

		// sort out the payload for this node
		payloadProto := &snapshotpb.Payload{}
		if perr := proto.Unmarshal(exportedNode.Value, payloadProto); perr != nil {
			return nil, perr
		}

		payloads = append(payloads, types.PayloadFromProto(payloadProto))

		exportedNode, err = exporter.Next()
	}

	if !errors.Is(err, iavl.ErrorExportDone) {
		return nil, fmt.Errorf("failed to export AVL tree: %w", err)
	}

	return payloads, nil
}

func (t *Tree) FindImmutableTreeByHeight(blockHeight uint64) (*iavl.ImmutableTree, error) {
	version, err := t.metadataDB.FindVersionByBlockHeight(blockHeight)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while looking for snapshot version: %w", err)
	}
	if version == -1 {
		return nil, fmt.Errorf("no snapshot found for block height %d", blockHeight)
	}

	return t.innerTree.GetImmutable(version)
}

func (t *Tree) ListLatestSnapshots(maxLengthOfSnapshotList uint64) ([]*tmtypes.Snapshot, error) {
	availableVersions := t.innerTree.AvailableVersions()
	numberOfAvailableVersions := len(availableVersions)

	listLength := maxLengthOfSnapshotList
	fromIndex := numberOfAvailableVersions - int(maxLengthOfSnapshotList) - 1

	// If negative, it means there is less versions than the maximum allowed, so
	// we start from 0.
	if fromIndex < 0 {
		fromIndex = 0
		listLength = uint64(numberOfAvailableVersions)
	}

	snapshotList := make([]*tmtypes.Snapshot, 0, listLength)

	snapshotCount := 0
	for i := fromIndex; i < numberOfAvailableVersions; i++ {
		version := int64(availableVersions[i])

		loadedSnapshot, err := t.metadataDB.Load(version)
		if err != nil {
			t.log.Error("could not load snapshot from the metadata database",
				logging.Int64("version", version),
				logging.Error(err),
			)
			// We ignore broken snapshot state.
			continue
		}
		snapshotList = append(snapshotList, loadedSnapshot)
		snapshotCount++
	}

	return snapshotList[0:snapshotCount], nil
}

func (t *Tree) AddSnapshot(s *types.Snapshot) error {
	importer, err := t.innerTree.Import(s.Meta.Version)
	if err != nil {
		return fmt.Errorf("could not initialize the AVL tree importer: %w", err)
	}
	defer importer.Close()

	// Convert slice into map for quick lookup.
	payloads := map[string]*types.Payload{}
	for _, pl := range s.Nodes {
		payloads[pl.TreeKey()] = pl
	}

	for _, n := range s.Meta.NodeHashes {
		var value []byte
		if n.IsLeaf {
			payload, ok := payloads[n.Key]
			if !ok {
				return fmt.Errorf("the payloads for key %q is missing from the snapshot: %w", n.Key, err)
			}
			value, err = proto.Marshal(payload.IntoProto())
			if err != nil {
				return fmt.Errorf("could not serialize the payload: %w", err)
			}
		} else {
			// The importer interprets any empty-non-nil slice as an actual empty
			// node. To ensure the importer correctly interprets it as "no value",
			// It has to be nil.
			// This has been made explicit for future reference.
			value = nil
		}

		exportedNode := &iavl.ExportNode{
			Key:   []byte(n.Key),
			Value: value,
			// This is the height of the node in the tree.
			Height: int8(n.Height),
			// This is the version of the node in the tree. It is incremented if
			// that node's value is updated.
			Version: n.Version,
		}

		if err := importer.Add(exportedNode); err != nil {
			return fmt.Errorf("could not import tree node: %w", err)
		}
	}

	if err := importer.Commit(); err != nil {
		return fmt.Errorf("could not finalize the snapshot import into the tree: %w", err)
	}

	snapshotAsTendermintFormat, err := s.ToTM()
	if err != nil {
		return fmt.Errorf("could not serialize the snapshot as Tendermint proto: %w", err)
	}

	if err := t.metadataDB.Save(t.innerTree.Version(), snapshotAsTendermintFormat); err != nil {
		return fmt.Errorf("could not save the snapshot metadata: %w", err)
	}

	return nil
}

func (t *Tree) Release() {
	if t.snapshotDB != nil {
		if err := t.snapshotDB.Close(); err != nil {
			t.log.Error("could not cleanly close the snapshot database", logging.Error(err))
		}
	}

	if t.metadataDB != nil {
		if err := t.metadataDB.Close(); err != nil {
			t.log.Error("could not cleanly close the metadata database", logging.Error(err))
		}
	}
}

func (t *Tree) initializeFromLocalStore() error {
	// This initialises the AVL tree based on the content of the database.
	// It's required to know the available versions, so we can perform look up
	// and clean up.
	// As a side effect, it will load the latest snapshot in the tree.
	if _, err := t.innerTree.Load(); err != nil {
		return fmt.Errorf("could not load local snapshots into the AVL tree: %w", err)
	}

	if err := t.removeOldSnapshots(); err != nil {
		return err
	}

	// Load the snapshot matching the specified block height. The specified block height
	// has to be a perfect match of the snapshot's block height, otherwise it fails.
	// If the block height is set to 0, it uses the latest snapshot, that has
	// already been loaded when calling (*MutableTree) Load(), at the
	// beginning of the method.
	if t.blockHeightToStartFrom > 0 {
		if err := t.loadTreeAtBlockHeight(t.blockHeightToStartFrom); err != nil {
			return err
		}
	}
	t.log.Info("Snapshot has been loaded", logging.Int64("version", t.innerTree.Version()))

	return nil
}

func (t *Tree) removeOldSnapshots() error {
	maxNumberOfSnapshotsToKeep := int(t.maxNumberOfSnapshotsToKeep)
	availableVersions := t.innerTree.AvailableVersions()
	currentNumberOfSnapshots := len(availableVersions)

	if currentNumberOfSnapshots > maxNumberOfSnapshotsToKeep {
		fromVersion := availableVersions[0]
		// The version defined by variable `toVersion` is excluded from the deletion.
		indexOfOldestVersionToKeep := currentNumberOfSnapshots - maxNumberOfSnapshotsToKeep
		toVersion := availableVersions[indexOfOldestVersionToKeep]
		if err := t.innerTree.DeleteVersionsRange(int64(fromVersion), int64(toVersion)); err != nil {
			// Based on the method documentation, this would only happen in the
			// presence of a programming error.
			return fmt.Errorf("could not remove old snapshots: %w", err)
		}

		t.log.Info("Old snapshots deleted",
			logging.Int("from-version", fromVersion),
			logging.Int("to-version", toVersion),
		)

		if err := t.metadataDB.DeleteRange(int64(fromVersion), int64(toVersion)); err != nil {
			return fmt.Errorf("could not remove old snapshots metadata: %w", err)
		}

		availableVersions = t.innerTree.AvailableVersions()
		currentNumberOfSnapshots = len(availableVersions)
	}

	if currentNumberOfSnapshots == 1 {
		t.log.Info("Single snapshot stored", logging.Int("version", availableVersions[0]))
	} else {
		t.log.Info("Multiple snapshots stored",
			logging.Int("from-version", availableVersions[0]),
			logging.Int("to-version", availableVersions[currentNumberOfSnapshots-1]),
		)
	}

	return nil
}

func (t *Tree) loadTreeAtBlockHeight(startHeight uint64) error {
	versionToLoad, err := t.metadataDB.FindVersionByBlockHeight(startHeight)
	if err != nil {
		return fmt.Errorf("an error occurred while looking for snapshot version: %w", err)
	}
	if versionToLoad == -1 {
		return fmt.Errorf("no snapshot found for block height %d", startHeight)
	}

	// Since it may reload a version anterior to the latest one, it has to reset
	// the tree as if `versionToLoad` was the latest known snapshot.
	// This helps to keep a clean snapshot database, and to prevent the upcoming
	// snapshots to step on these older snapshots, and cause mayhem.
	// The mayhem would originate from the underlying AVL tree library that will
	// save the upcoming snapshot as one that comes right after the loaded one,
	// without accounting for the ones that already exist.
	if _, err := t.innerTree.LoadVersionForOverwriting(versionToLoad); err != nil {
		return fmt.Errorf("could not load snapshot with version %d in AVL tree: %w", versionToLoad, err)
	}

	return nil
}

func (t *Tree) SaveVersion() error {
	_, _, err := t.innerTree.SaveVersion()
	if err != nil {
		return fmt.Errorf("could not save the working tree: %w", err)
	}

	availableVersion := t.innerTree.AvailableVersions()
	numberOfAvailableVersion := uint64(len(availableVersion))

	if numberOfAvailableVersion > t.maxNumberOfSnapshotsToKeep {
		versionToDelete := int64(availableVersion[0])
		if err := t.innerTree.DeleteVersion(versionToDelete); err != nil {
			t.log.Error("Could not remove old snapshot ",
				logging.Int64("version", versionToDelete),
				logging.Error(err),
			)
		} else {
			t.log.Info("Old snapshot deleted", logging.Int64("version", versionToDelete))
		}

		if err := t.metadataDB.Delete(versionToDelete); err != nil {
			t.log.Error("Could not remove old snapshot metadata",
				logging.Int64("version", versionToDelete),
				logging.Error(err),
			)
		} else {
			t.log.Info("Old snapshot metadata deleted", logging.Int64("version", versionToDelete))
		}
	}

	immutableTree, err := t.innerTree.GetImmutable(t.innerTree.Version())
	if err != nil {
		return fmt.Errorf("could not generate immutable tree: %w", err)
	}

	snapshot, err := types.SnapshotFromTree(immutableTree)
	if err != nil {
		return fmt.Errorf("could not serialize the snapshot from tree: %w", err)
	}

	tendermintSnapshot, err := snapshot.ToTM()
	if err != nil {
		return fmt.Errorf("could not serialize the snapshot as Tendermint proto: %w", err)
	}

	if err := t.metadataDB.Save(t.innerTree.Version(), tendermintSnapshot); err != nil {
		return fmt.Errorf("could not save the snapshot to metadata database: %w", err)
	}

	return nil
}

func New(log *logging.Logger, opts ...Options) (*Tree, error) {
	tree := &Tree{
		log: log,

		maxNumberOfSnapshotsToKeep: 10,
		blockHeightToStartFrom:     0,
	}

	for _, opt := range opts {
		if err := opt(tree); err != nil {
			// If it fails initialization, any allocated resources is released.
			tree.Release()
			return nil, err
		}
	}

	if tree.snapshotDB == nil || tree.metadataDB == nil {
		panic("the databases have not been initialize")
	}

	innerTree, err := iavl.NewMutableTree(tree.snapshotDB, 0, false)
	if err != nil {
		return nil, fmt.Errorf("could not initialize the AVL tree: %w", err)
	}
	tree.innerTree = innerTree

	// TODO: At this point, we should ensure the state of the metadata database
	//  is as consistent as possible with snapshots database. This will lower
	//  the probability of errors.

	if tree.metadataDB.IsEmpty() {
		// There is no metadata, so we assume there is no snapshots to load from.
		return tree, nil
	}

	if err := tree.initializeFromLocalStore(); err != nil {
		return nil, fmt.Errorf("could not load local snapshots into the tree: %w", err)
	}

	return tree, nil
}
