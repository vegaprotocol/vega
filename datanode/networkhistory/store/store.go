package store

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"code.vegaprotocol.io/vega/datanode/metrics"

	"github.com/ipfs/kubo/repo"

	"github.com/ipfs/kubo/core/corerepo"

	"code.vegaprotocol.io/vega/datanode/networkhistory/fsutil"
	"code.vegaprotocol.io/vega/datanode/networkhistory/snapshot"
	"code.vegaprotocol.io/vega/logging"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/repo/fsrepo"

	"github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/ipfs/kubo/config"
	serialize "github.com/ipfs/kubo/config/serialize"
	"github.com/ipfs/kubo/core"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/ipfs/kubo/core/corehttp"
	"github.com/ipfs/kubo/plugin/loader"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/prometheus/client_golang/prometheus"

	ipfslogging "github.com/ipfs/go-log"
)

const segmentMetaDataFile = "metadata.json"

var ErrSegmentNotFound = errors.New("segment not found")

type index interface {
	Get(height int64) (SegmentIndexEntry, error)
	Add(metaData SegmentIndexEntry) error
	Remove(indexEntry SegmentIndexEntry) error
	ListAllEntriesOldestFirst() ([]SegmentIndexEntry, error)
	GetHighestBlockHeightEntry() (SegmentIndexEntry, error)
	Close() error
}

type SegmentMetaData struct {
	HeightFrom               int64
	HeightTo                 int64
	ChainID                  string
	PreviousHistorySegmentID string
}

func (m SegmentMetaData) GetFromHeight() int64 {
	return m.HeightFrom
}

func (m SegmentMetaData) GetToHeight() int64 {
	return m.HeightTo
}

func (i SegmentIndexEntry) GetPreviousHistorySegmentId() string {
	return i.PreviousHistorySegmentID
}

type SegmentIndexEntry struct {
	SegmentMetaData
	HistorySegmentID string
}

func (i SegmentIndexEntry) GetHistorySegmentId() string {
	return i.HistorySegmentID
}

type Store struct {
	log      *logging.Logger
	cfg      Config
	identity config.Identity
	ipfsAPI  icore.CoreAPI
	ipfsNode *core.IpfsNode
	ipfsRepo repo.Repo
	index    index
	swarmKey string

	indexPath  string
	stagingDir string
	ipfsPath   string
}

// This global var is to prevent IPFS plugins being loaded twice because IPFS uses a dependency injection framework that
// has global state which results in an error if ipfs plugins are loaded twice.  In practice this is currently only an
// issue when running tests as we only have one IPFS node instance when running datanode.
var plugins *loader.PluginLoader

func New(ctx context.Context, log *logging.Logger, chainID string, cfg Config, networkHistoryHome string, wipeOnStartup bool) (*Store, error) {
	if log.IsDebug() {
		ipfslogging.SetDebugLogging()
	}

	networkHistoryStorePath := filepath.Join(networkHistoryHome, "store")

	p := &Store{
		log:        log,
		cfg:        cfg,
		indexPath:  filepath.Join(networkHistoryStorePath, "index"),
		stagingDir: filepath.Join(networkHistoryStorePath, "staging"),
		ipfsPath:   filepath.Join(networkHistoryStorePath, "ipfs"),
	}

	err := p.setupPaths(networkHistoryStorePath, wipeOnStartup)
	if err != nil {
		return nil, fmt.Errorf("failed to setup paths:%w", err)
	}

	p.index, err = NewIndex(p.indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create index:%w", err)
	}

	if len(chainID) == 0 {
		return nil, fmt.Errorf("chain ID must be set")
	}

	if len(cfg.PeerID) == 0 || len(cfg.PrivKey) == 0 {
		return nil, fmt.Errorf("the ipfs peer id and priv key must be set")
	}

	p.identity = config.Identity{
		PeerID:  cfg.PeerID,
		PrivKey: cfg.PrivKey,
	}

	log.Infof("starting network history store with ipfs Peer Id:%s", p.identity.PeerID)

	if plugins == nil {
		plugins, err = loadPlugins(p.ipfsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load ipfs plugins:%w", err)
		}
	}

	log.Debugf("ipfs swarm port:%d", cfg.SwarmPort)
	ipfsCfg, err := createIpfsNodeConfiguration(p.log, p.identity, cfg.BootstrapPeers,
		cfg.SwarmPort)

	log.Debugf("ipfs bootstrap peers:%v", ipfsCfg.Bootstrap)

	if err != nil {
		return nil, fmt.Errorf("failed to create ipfs node configuration:%w", err)
	}

	p.swarmKey = cfg.GetSwarmKey(log, chainID)

	p.ipfsNode, p.ipfsRepo, err = createIpfsNode(ctx, log, p.ipfsPath, ipfsCfg, p.swarmKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create ipfs node:%w", err)
	}

	if p.ipfsNode.PNetFingerprint != nil {
		log.Infof("Swarm is limited to private network of peers with the fingerprint %x", p.ipfsNode.PNetFingerprint)
	}

	p.ipfsAPI, err = coreapi.NewCoreAPI(p.ipfsNode)

	if err != nil {
		return nil, fmt.Errorf("failed to create ipfs api:%w", err)
	}

	if err = setupMetrics(p.ipfsNode); err != nil {
		return nil, fmt.Errorf("failed to setup metrics:%w", err)
	}

	return p, nil
}

func (p *Store) Stop() {
	p.log.Info("Cleaning up network history store")
	if p.ipfsNode != nil {
		p.log.Info("Closing IPFS node")
		_ = p.ipfsNode.Close()
	}

	if p.index != nil {
		p.log.Info("Closing LevelDB")
		_ = p.index.Close()
	}
}

func (p *Store) GetSwarmKey() string {
	return p.swarmKey
}

func (p *Store) GetPeerAddrs() []ma.Multiaddr {
	addrs := make([]ma.Multiaddr, 0, 10)

	thisNode := p.ipfsNode.PeerHost.Network().LocalPeer()
	peers := p.ipfsNode.PeerHost.Network().Peers()

	for _, peer := range peers {
		if peer == thisNode {
			continue
		}

		connections := p.ipfsNode.PeerHost.Network().ConnsToPeer(peer)
		for _, conn := range connections {
			addrs = append(addrs, conn.RemoteMultiaddr())
		}
	}

	return addrs
}

func (p *Store) ResetIndex() error {
	err := os.RemoveAll(p.indexPath)
	if err != nil {
		return fmt.Errorf("failed to remove index path:%w", err)
	}

	err = os.MkdirAll(p.indexPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create index path:%w", err)
	}

	p.index, err = NewIndex(p.indexPath)
	if err != nil {
		return fmt.Errorf("failed to create index:%w", err)
	}

	return nil
}

func (p *Store) GetPeerID() string {
	return p.identity.PeerID
}

func (p *Store) ConnectedToPeer(peerIDStr string) (bool, error) {
	p.ipfsNode.PeerHost.Network().Conns()

	for _, pr := range p.ipfsNode.PeerHost.Network().Peers() {
		if pr.String() == peerIDStr {
			return true, nil
		}
	}
	return false, nil
}

func (p *Store) AddSnapshotData(ctx context.Context, historySnapshot snapshot.History, currentState snapshot.CurrentState,
	sourceDir string,
) (err error) {
	historyID := fmt.Sprintf("%s-%d-%d", historySnapshot.ChainID, historySnapshot.HeightFrom, historySnapshot.HeightTo)

	p.log.Infof("adding history %s", historyID)

	historyStagingDir := filepath.Join(p.stagingDir, historyID)
	historyStagingSnapshotDir := filepath.Join(historyStagingDir, "snapshotData")

	compressedCurrentStateSnapshotFile := filepath.Join(sourceDir, currentState.CompressedFileName())
	compressedHistorySnapshotFile := filepath.Join(sourceDir, historySnapshot.CompressedFileName())

	defer func() {
		_ = os.RemoveAll(historyStagingDir)
		_ = os.RemoveAll(compressedCurrentStateSnapshotFile)
		_ = os.RemoveAll(compressedHistorySnapshotFile)
	}()

	historySegment := SegmentMetaData{
		HeightFrom:               historySnapshot.HeightFrom,
		HeightTo:                 historySnapshot.HeightTo,
		ChainID:                  historySnapshot.ChainID,
		PreviousHistorySegmentID: "",
	}

	historySegment.PreviousHistorySegmentID, err = p.getPreviousHistorySegmentID(historySegment)
	if err != nil {
		if !errors.Is(err, ErrSegmentNotFound) {
			return fmt.Errorf("failed to get previous history segment id:%w", err)
		}
	}

	if err = os.MkdirAll(historyStagingDir, fs.ModePerm); err != nil {
		return fmt.Errorf("failed to make history staging directory:%w", err)
	}

	if err = os.MkdirAll(historyStagingSnapshotDir, fs.ModePerm); err != nil {
		return fmt.Errorf("failed to make history staging snapshot directory:%w", err)
	}

	metaDataBytes, err := json.Marshal(historySegment)
	if err != nil {
		return fmt.Errorf("failed to marshal meta data:%w", err)
	}

	if err = os.WriteFile(filepath.Join(historyStagingSnapshotDir, segmentMetaDataFile), metaDataBytes, fs.ModePerm); err != nil {
		return fmt.Errorf("failed to write meta data:%w", err)
	}

	err = os.Rename(compressedCurrentStateSnapshotFile, filepath.Join(historyStagingSnapshotDir, currentState.CompressedFileName()))
	if err != nil {
		return fmt.Errorf("failed to move currentState into publish staging directory:%w", err)
	}

	err = os.Rename(compressedHistorySnapshotFile, filepath.Join(historyStagingSnapshotDir, historySnapshot.CompressedFileName()))
	if err != nil {
		return fmt.Errorf("failed to move history into publish staging directory:%w", err)
	}

	tarFileName := filepath.Join(historyStagingDir, historyID+".tar")
	tarFile, err := os.Create(tarFileName)
	if err != nil {
		return fmt.Errorf("failed to create tar file:%w", err)
	}

	if err = fsutil.TarDirectoryWithDeterministicHeader(tarFile, 0, historyStagingSnapshotDir); err != nil {
		return fmt.Errorf("failed to create tar staging directory:%w", err)
	}

	contentID, err := p.addHistorySegment(ctx, tarFileName, historySegment)
	if err != nil {
		return fmt.Errorf("failed to add file:%w", err)
	}

	p.log.Info("finished adding history to network history store",
		logging.String("history segment id", contentID.String()),
		logging.String("chain id", historySegment.ChainID),
		logging.Int64("from height", historySegment.HeightFrom),
		logging.Int64("to height", historySegment.HeightTo),
		logging.String("previous history segment id", historySegment.PreviousHistorySegmentID),
	)

	segments, err := p.removeOldHistorySegments(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove old history segments:%s", err)
	}
	p.log.Infof("removed %d old history segments", len(segments))

	ipfsSize, err := p.ipfsRepo.GetStorageUsage(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the ipfs storage usage: %w", err)
	}
	metrics.SetNetworkHistoryIpfsStoreBytes(float64(ipfsSize))

	return nil
}

func (p *Store) GetHighestBlockHeightEntry() (SegmentIndexEntry, error) {
	entry, err := p.index.GetHighestBlockHeightEntry()
	if err != nil {
		if errors.Is(err, ErrIndexEntryNotFound) {
			return SegmentIndexEntry{}, ErrSegmentNotFound
		}

		return SegmentIndexEntry{}, fmt.Errorf("failed to get highest block height entry from index:%w", err)
	}

	return entry, nil
}

func (p *Store) ListAllIndexEntriesOldestFirst() ([]SegmentIndexEntry, error) {
	return p.index.ListAllEntriesOldestFirst()
}

func (p *Store) CopySnapshotDataIntoDir(ctx context.Context, toHeight int64, targetDir string) (currentStateSnapshot snapshot.CurrentState,
	historySnapshot snapshot.History, err error,
) {
	defer func() {
		deferErr := fsutil.RemoveAllFromDirectoryIfExists(p.stagingDir)
		if err == nil {
			err = deferErr
		}
	}()

	err = fsutil.RemoveAllFromDirectoryIfExists(p.stagingDir)
	if err != nil {
		return snapshot.CurrentState{}, snapshot.History{}, fmt.Errorf("failed to empty staging directory:%w", err)
	}

	err = p.extractHistorySegmentToStagingArea(ctx, toHeight)
	if err != nil {
		return snapshot.CurrentState{}, snapshot.History{}, fmt.Errorf("failed to extract history segment to staging area:%w", err)
	}

	currentStateSnapshot, historySnapshot, err = p.getSnapshotsFromStagingArea(toHeight)
	if err != nil {
		return snapshot.CurrentState{}, snapshot.History{}, fmt.Errorf("failed to get snapshots from staging area:%w", err)
	}

	if err = moveSnapshotData(currentStateSnapshot, historySnapshot, p.stagingDir, targetDir); err != nil {
		return snapshot.CurrentState{}, snapshot.History{}, fmt.Errorf("failed to move snapshots from staging area to target directory %s:%w", targetDir, err)
	}

	return currentStateSnapshot, historySnapshot, nil
}

func setupMetrics(ipfsNode *core.IpfsNode) error {
	err := prometheus.Register(&corehttp.IpfsNodeCollector{Node: ipfsNode})
	if err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			return fmt.Errorf("failed to initialise IPFS metrics:%w", err)
		}
	}

	return nil
}

func (p *Store) getPreviousHistorySegmentID(history SegmentMetaData) (string, error) {
	var err error
	var previousHistorySegment SegmentIndexEntry
	if history.HeightFrom > 0 {
		height := history.HeightFrom - 1
		previousHistorySegment, err = p.index.Get(height)
		if errors.Is(err, ErrIndexEntryNotFound) {
			return "", ErrSegmentNotFound
		}

		if err != nil {
			return "", fmt.Errorf("failed to get index entry for height:%w", err)
		}
	}
	return previousHistorySegment.HistorySegmentID, nil
}

func (p *Store) addHistorySegment(ctx context.Context, historySegmentFile string, fileIndexEntry SegmentMetaData) (cid.Cid, error) {
	contentID, err := p.addFileToIpfs(ctx, historySegmentFile)
	if err != nil {
		return cid.Cid{}, fmt.Errorf("failed to add history segement %s to ipfs:%w", historySegmentFile, err)
	}

	if err = p.index.Add(SegmentIndexEntry{
		SegmentMetaData:  fileIndexEntry,
		HistorySegmentID: contentID.String(),
	}); err != nil {
		return cid.Cid{}, fmt.Errorf("failed to update meta data store:%w", err)
	}
	return contentID, nil
}

func (p *Store) setupPaths(networkHistoryStorePath string, wipeOnStartup bool) error {
	if wipeOnStartup {
		err := os.RemoveAll(networkHistoryStorePath)
		if err != nil {
			return fmt.Errorf("failed to remove dir:%w", err)
		}
	}

	err := os.MkdirAll(p.indexPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create index path:%w", err)
	}

	err = os.MkdirAll(p.stagingDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create staging path:%w", err)
	}

	return nil
}

func moveSnapshotData(currentStateSnapshot snapshot.CurrentState, historySnapshot snapshot.History, sourceDir, targetDir string) error {
	if err := os.Rename(filepath.Join(sourceDir, currentStateSnapshot.CompressedFileName()),
		filepath.Join(targetDir, currentStateSnapshot.CompressedFileName())); err != nil {
		return fmt.Errorf("failed to move current state snapshot:%w", err)
	}

	if err := os.Rename(filepath.Join(sourceDir, historySnapshot.CompressedFileName()), filepath.Join(targetDir, historySnapshot.CompressedFileName())); err != nil {
		return fmt.Errorf("failed to move history snapshot:%w", err)
	}

	return nil
}

func (p *Store) getSnapshotsFromStagingArea(toHeight int64) (snapshot.CurrentState, snapshot.History, error) {
	_, currentStateSnapshots, err := snapshot.GetCurrentStateSnapshots(p.stagingDir)
	if err != nil {
		return snapshot.CurrentState{}, snapshot.History{}, fmt.Errorf("failed to get current state snapshot from staging area:%w", err)
	}

	if len(currentStateSnapshots) != 1 {
		return snapshot.CurrentState{}, snapshot.History{}, fmt.Errorf("expected 1 current state snapshot in staging area, found %d", len(currentStateSnapshots))
	}

	currentStateSnapshot, ok := currentStateSnapshots[toHeight]
	if !ok {
		return snapshot.CurrentState{}, snapshot.History{}, fmt.Errorf("failed to find current state snapshot for height %d in the staging area", toHeight)
	}

	_, historySnapshots, err := snapshot.GetHistorySnapshots(p.stagingDir)
	if err != nil {
		return snapshot.CurrentState{}, snapshot.History{}, fmt.Errorf("failed to get history snapshot from staging area:%w", err)
	}

	if len(historySnapshots) != 1 {
		return snapshot.CurrentState{}, snapshot.History{}, fmt.Errorf("expected 1 history state snapshot in staging area, found %d", len(historySnapshots))
	}
	historySnapshot := historySnapshots[0]

	return currentStateSnapshot, historySnapshot, nil
}

func (p *Store) extractHistorySegmentToStagingArea(ctx context.Context, toHeight int64) error {
	historySegmentPath, err := p.getHistorySegmentForHeight(ctx, toHeight, p.stagingDir)
	if err != nil {
		return fmt.Errorf("failed to get history segment:%w", err)
	}

	stagingFile, err := os.Open(historySegmentPath)
	if err != nil {
		return fmt.Errorf("failed to open staging file:%w", err)
	}
	defer func() { _ = stagingFile.Close() }()

	err = fsutil.UntarFile(stagingFile, p.stagingDir)
	if err != nil {
		return fmt.Errorf("failed to untar staging file:%w", err)
	}
	return nil
}

func (p *Store) getHistorySegmentForHeight(ctx context.Context, toHeight int64, toDir string) (pathToSegment string, err error) {
	indexEntry, err := p.index.Get(toHeight)
	if err != nil {
		return "", fmt.Errorf("failed to get index entry for height:%d:%w", toHeight, err)
	}
	segmentFileName := fmt.Sprintf("%s-%d-%d.tar", indexEntry.ChainID, indexEntry.HeightFrom, indexEntry.HeightTo)

	historySegmentID := indexEntry.HistorySegmentID
	pathToSegment = filepath.Join(toDir, segmentFileName)
	err = p.CopyHistorySegmentToFile(ctx, historySegmentID, pathToSegment)
	if err != nil {
		return "", fmt.Errorf("failed to get history segment:%w", err)
	}
	return pathToSegment, nil
}

func (p *Store) CopyHistorySegmentToFile(ctx context.Context, historySegmentID string, targetFile string) error {
	ipfsCid, err := cid.Parse(historySegmentID)
	if err != nil {
		return fmt.Errorf("failed to parse history segment id:%w", err)
	}

	ipfsFile, err := p.ipfsAPI.Unixfs().Get(ctx, path.IpfsPath(ipfsCid))
	if err != nil {
		return fmt.Errorf("failed to get ipfs file:%w", err)
	}

	if err = files.WriteTo(ipfsFile, targetFile); err != nil {
		return fmt.Errorf("failed to write to staging file:%w", err)
	}
	return nil
}

func (p *Store) removeOldHistorySegments(ctx context.Context) ([]SegmentIndexEntry, error) {
	latestSegment, err := p.index.GetHighestBlockHeightEntry()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest segment:%w", err)
	}

	entries, err := p.index.ListAllEntriesOldestFirst()
	if err != nil {
		return nil, fmt.Errorf("failed to list all entries:%w", err)
	}

	var removedSegments []SegmentIndexEntry
	for _, segment := range entries {
		if segment.HeightTo < (latestSegment.HeightTo - p.cfg.HistoryRetentionBlockSpan) {
			err = p.unpinSegment(ctx, segment)
			if err != nil {
				return nil, fmt.Errorf("failed to unpin segment:%w", err)
			}

			p.index.Remove(segment)

			removedSegments = append(removedSegments, segment)
		} else {
			break
		}
	}

	if len(removedSegments) > 0 {
		// The GarbageCollect method is async
		err = corerepo.GarbageCollect(p.ipfsNode, ctx)

		// Do not want to return before the GC is done as adding new data to the node whilst GC is running is not permitted
		unlocker := p.ipfsNode.GCLocker.GCLock(ctx)
		defer unlocker.Unlock(ctx)

		if err != nil {
			return nil, fmt.Errorf("failed to garbage collect ipfs repo")
		}
	}

	return removedSegments, nil
}

func (p *Store) FetchHistorySegment(ctx context.Context, historySegmentID string) (SegmentIndexEntry, error) {
	historySegment := filepath.Join(p.stagingDir, "historySegment.tar")

	err := os.RemoveAll(historySegment)
	if err != nil {
		return SegmentIndexEntry{}, fmt.Errorf("failed to remove existing history segment tar: %w", err)
	}

	contentID, err := cid.Parse(historySegmentID)
	if err != nil {
		return SegmentIndexEntry{}, fmt.Errorf("failed to parse snapshotId into CID:%w", err)
	}

	rootNodeFile, err := p.ipfsAPI.Unixfs().Get(ctx, path.IpfsPath(contentID))
	if err != nil {
		connInfo, swarmError := p.ipfsAPI.Swarm().Peers(ctx)
		if swarmError != nil {
			return SegmentIndexEntry{}, fmt.Errorf("failed to get peers: %w", err)
		}

		peerAddrs := ""
		for _, peer := range connInfo {
			peerAddrs += fmt.Sprintf(",%s", peer.Address())
		}

		return SegmentIndexEntry{}, fmt.Errorf("could not get file with CID, connected peer addresses %s: %w", peerAddrs, err)
	}

	err = files.WriteTo(rootNodeFile, historySegment)
	if err != nil {
		return SegmentIndexEntry{}, fmt.Errorf("could not write out the fetched history segment: %w", err)
	}

	tarFile, err := os.Open(historySegment)
	if err != nil {
		return SegmentIndexEntry{}, fmt.Errorf("failed to open history segment: %w", err)
	}
	defer func() { _ = tarFile.Close() }()

	historySegmentDir := filepath.Join(p.stagingDir, "historySegment")
	err = os.RemoveAll(historySegmentDir)
	if err != nil {
		return SegmentIndexEntry{}, fmt.Errorf("failed to remove exisiting history segment dir: %w", err)
	}

	err = os.Mkdir(historySegmentDir, os.ModePerm)
	if err != nil {
		return SegmentIndexEntry{}, fmt.Errorf("failed to create history segment dir: %w", err)
	}
	err = fsutil.UntarFile(tarFile, historySegmentDir)
	if err != nil {
		return SegmentIndexEntry{}, fmt.Errorf("failed to untar history segment:%w", err)
	}

	indexEntryBytes, err := os.ReadFile(filepath.Join(historySegmentDir, segmentMetaDataFile))
	if err != nil {
		return SegmentIndexEntry{}, fmt.Errorf("failed to read index entry:%w", err)
	}

	var fileIndex SegmentMetaData
	if err = json.Unmarshal(indexEntryBytes, &fileIndex); err != nil {
		return SegmentIndexEntry{}, fmt.Errorf("failed to unmarshal index entry:%w", err)
	}

	indexEntry := SegmentIndexEntry{
		SegmentMetaData:  fileIndex,
		HistorySegmentID: historySegmentID,
	}

	if err = p.index.Add(indexEntry); err != nil {
		return SegmentIndexEntry{}, fmt.Errorf("failed to add index entry:%w", err)
	}

	return indexEntry, nil
}

func createIpfsNodeConfiguration(log *logging.Logger, identity config.Identity, bootstrapPeers []string, swarmPort int) (*config.Config, error) {
	cfg, err := config.InitWithIdentity(identity)

	// Don't try and do local node discovery with mDNS; we're probably on the internet if running
	// for real, and in tests we explicitly want to set up our network by specifying bootstrap peers
	cfg.Discovery.MDNS.Enabled = false

	if err != nil {
		return nil, fmt.Errorf("failed to initiliase ipfs config:%w", err)
	}

	const ipfsConfigDefaultSwarmPort = "4001"
	updatedSwarmAddrs := make([]string, 0, 10)
	for _, addr := range cfg.Addresses.Swarm {
		// Exclude ip6 addresses cause hang on lookup
		if !strings.Contains(addr, "/ip6/") {
			updatedSwarmAddrs = append(updatedSwarmAddrs, strings.ReplaceAll(addr, ipfsConfigDefaultSwarmPort, strconv.Itoa(swarmPort)))
		}
	}

	cfg.Addresses.Swarm = updatedSwarmAddrs
	cfg.Bootstrap = bootstrapPeers

	prettyCfgJSON, _ := json.MarshalIndent(cfg, "", "  ")
	log.Debugf("IPFS Node Config:\n%s", prettyCfgJSON)

	return cfg, nil
}

func updateRepoConfig(path string, conf *config.Config) error {
	configFilename, err := config.Filename(path, "")
	if err != nil {
		return fmt.Errorf("failed to get the configuration file path:%w", err)
	}

	if err = serialize.WriteConfigFile(configFilename, conf); err != nil {
		return fmt.Errorf("failed to write the config file:%w", err)
	}

	return nil
}

func loadPlugins(externalPluginsPath string) (*loader.PluginLoader, error) {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return nil, fmt.Errorf("error loading plugins: %s", err)
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return nil, fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return nil, fmt.Errorf("error injecting plugins: %s", err)
	}

	return plugins, nil
}

func generateSwarmKeyFile(swarmKey string, repoPath string) error {
	file, err := os.Create(filepath.Join(repoPath, "swarm.key"))
	defer func() { _ = file.Close() }()
	if err != nil {
		return fmt.Errorf("failed to create swarm key file:%w", err)
	}

	key := make([]byte, 32)

	copy(key, swarmKey)
	hx := hex.EncodeToString(key)

	_, err = io.WriteString(file, fmt.Sprintf("/key/swarm/psk/1.0.0/\n/base16/\n%s\n", hx))

	if err != nil {
		return fmt.Errorf("failed to write to file:%w", err)
	}

	return nil
}

func createNode(ctx context.Context, log *logging.Logger, repoPath string) (*core.IpfsNode, repo.Repo, error) {
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open ipfs repo:%w", err)
	}

	// Construct the node

	nodeOptions := &core.BuildCfg{
		Online:    true,
		Permanent: true,
		Routing:   libp2p.DHTOption, // This option sets the node to be a full DHT node (both fetching and storing DHT Records)
		Repo:      repo,
	}

	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create new node:%w", err)
	}

	printSwarmAddrs(node, log)

	// Attach the Core API to the constructed node
	return node, repo, nil
}

func printSwarmAddrs(node *core.IpfsNode, log *logging.Logger) {
	if !node.IsOnline {
		log.Debugf("Swarm not listening, running in offline mode.")
		return
	}

	ifaceAddrs, err := node.PeerHost.Network().InterfaceListenAddresses()
	if err != nil {
		log.Debugf("failed to read listening addresses: %s", err)
	}
	lisAddrs := make([]string, len(ifaceAddrs))
	for i, addr := range ifaceAddrs {
		lisAddrs[i] = addr.String()
	}
	sort.Strings(lisAddrs)
	for _, addr := range lisAddrs {
		log.Debugf("Swarm listening on %s\n", addr)
	}

	nodePhostAddrs := node.PeerHost.Addrs()
	addrs := make([]string, len(nodePhostAddrs))
	for i, addr := range nodePhostAddrs {
		addrs[i] = addr.String()
	}
	sort.Strings(addrs)
	for _, addr := range addrs {
		log.Debugf("Swarm announcing %s\n", addr)
	}
}

func createIpfsNode(ctx context.Context, log *logging.Logger, repoPath string,
	cfg *config.Config, swarmKey string,
) (*core.IpfsNode, repo.Repo, error) {
	// Only inits the repo if it does not already exist
	err := fsrepo.Init(repoPath, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialise ipfs configuration:%w", err)
	}

	// Update to take account of any new bootstrap nodes
	err = updateRepoConfig(repoPath, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update ipfs configuration:%w", err)
	}

	err = generateSwarmKeyFile(swarmKey, repoPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate swarm key file:%w", err)
	}

	return createNode(ctx, log, repoPath)
}

func (p *Store) addFileToIpfs(ctx context.Context, path string) (cid.Cid, error) {
	file, err := os.Open(path)
	if err != nil {
		return cid.Cid{}, err
	}
	defer func() { _ = file.Close() }()

	st, err := file.Stat()
	if err != nil {
		return cid.Cid{}, err
	}

	f, err := files.NewReaderPathFile(path, file, st)
	if err != nil {
		return cid.Cid{}, err
	}

	fileCid, err := p.ipfsAPI.Unixfs().Add(ctx, f)
	if err != nil {
		return cid.Cid{}, fmt.Errorf("failed to add file: %s", err)
	}

	err = p.ipfsAPI.Pin().Add(ctx, fileCid)
	if err != nil {
		return cid.Cid{}, fmt.Errorf("failed to pin file: %s", err)
	}
	return fileCid.Cid(), nil
}

func (p *Store) unpinSegment(ctx context.Context, segment SegmentIndexEntry) error {
	contentID, err := cid.Decode(segment.HistorySegmentID)
	if err != nil {
		return fmt.Errorf("failed to decode history segment id:%w", err)
	}

	path := path.IpfsPath(contentID)

	if err = p.ipfsAPI.Pin().Rm(ctx, path); err != nil {
		return fmt.Errorf("failed to unpin segment:%w", err)
	}

	return nil
}
