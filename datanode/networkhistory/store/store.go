package store

import (
	"archive/zip"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/ipfs/kubo/core/node/libp2p/fd"
	"github.com/libp2p/go-libp2p/core/peer"

	"code.vegaprotocol.io/vega/datanode/metrics"

	"github.com/ipfs/kubo/repo"

	"github.com/ipfs/kubo/core/corerepo"

	"code.vegaprotocol.io/vega/datanode/networkhistory/segment"
	"code.vegaprotocol.io/vega/libs/memory"
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
	Get(height int64) (segment.Full, error)
	Add(metaData segment.Full) error
	Remove(indexEntry segment.Full) error
	ListAllEntriesOldestFirst() (segment.Segments[segment.Full], error)
	GetHighestBlockHeightEntry() (segment.Full, error)
	Close() error
}

type IpfsNode struct {
	IpfsId peer.ID
	Addr   ma.Multiaddr
}

func (i IpfsNode) IpfsAddress() (ma.Multiaddr, error) {
	ipfsProtocol, err := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", i.IpfsId))
	if err != nil {
		return nil, fmt.Errorf("failed to create new p2p multi address: %w", err)
	}

	return i.Addr.Encapsulate(ipfsProtocol), nil
}

type PeerConnection struct {
	Local  IpfsNode
	Remote IpfsNode
}

type Store struct {
	log          *logging.Logger
	cfg          Config
	identity     config.Identity
	ipfsAPI      icore.CoreAPI
	ipfsNode     *core.IpfsNode
	ipfsRepo     repo.Repo
	index        index
	swarmKeySeed string
	swarmKey     string

	indexPath  string
	stagingDir string
	ipfsPath   string
}

// This global var is to prevent IPFS plugins being loaded twice because IPFS uses a dependency injection framework that
// has global state which results in an error if ipfs plugins are loaded twice.  In practice this is currently only an
// issue when running tests as we only have one IPFS node instance when running datanode.
var plugins *loader.PluginLoader

func New(ctx context.Context, log *logging.Logger, chainID string, cfg Config, networkHistoryHome string,
	wipeOnStartup bool, maxMemoryPercent uint8,
) (*Store, error) {
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

	idxLog := log.With(logging.String("component", "index"))
	p.index, err = NewIndex(p.indexPath, idxLog)
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

	p.swarmKeySeed = cfg.GetSwarmKeySeed(log, chainID)

	p.ipfsNode, p.ipfsRepo, p.swarmKey, err = createIpfsNode(ctx, log, p.ipfsPath, ipfsCfg, p.swarmKeySeed, maxMemoryPercent)
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
		if err := p.ipfsNode.Close(); err != nil {
			p.log.Error("Failed to close IPFS node", logging.Error(err))
		}
	}

	if p.index != nil {
		if err := p.index.Close(); err != nil {
			p.log.Error("Failed to close LevelDB:%s", logging.Error(err))
		}
		p.log.Info("LevelDB closed")
	}
}

func (p *Store) GetSwarmKey() string {
	return p.swarmKey
}

func (p *Store) GetSwarmKeySeed() string {
	return p.swarmKeySeed
}

func (p *Store) GetLocalNode() (IpfsNode, error) {
	localNodeMultiAddress, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", p.cfg.SwarmPort))
	if err != nil {
		return IpfsNode{}, fmt.Errorf("failed to create default multi addr: %w", err)
	}

	localNode := IpfsNode{
		IpfsId: p.ipfsNode.PeerHost.Network().LocalPeer(),
		Addr:   localNodeMultiAddress,
	}

	connectedPeers := p.GetConnectedPeers()
	if err != nil {
		return IpfsNode{}, fmt.Errorf("failed to get connected peers: %w", err)
	}

	tcpProtocol := ma.ProtocolWithName("tcp")
	for _, cp := range connectedPeers {
		port, err := cp.Local.Addr.ValueForProtocol(tcpProtocol.Code)
		if err == nil {
			if port == strconv.Itoa(p.cfg.SwarmPort) {
				localNode.Addr = cp.Local.Addr
				break
			}
		}
	}

	return localNode, nil
}

func (p *Store) GetConnectedPeers() []PeerConnection {
	peerConnections := make([]PeerConnection, 0, 10)

	thisNode := p.ipfsNode.PeerHost.Network().LocalPeer()
	peers := p.ipfsNode.PeerHost.Network().Peers()

	for _, peer := range peers {
		if peer == thisNode {
			continue
		}

		connections := p.ipfsNode.PeerHost.Network().ConnsToPeer(peer)
		for _, conn := range connections {
			peerConnections = append(peerConnections, PeerConnection{
				Local: IpfsNode{
					IpfsId: conn.LocalPeer(),
					Addr:   conn.LocalMultiaddr(),
				},
				Remote: IpfsNode{
					IpfsId: conn.RemotePeer(),
					Addr:   conn.RemoteMultiaddr(),
				},
			})
		}
	}

	return peerConnections
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

	idxLog := p.log.With(logging.String("component", "index"))
	p.index, err = NewIndex(p.indexPath, idxLog)
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

func (p *Store) AddSnapshotData(ctx context.Context, s segment.Unpublished) (err error) {
	historyID := fmt.Sprintf("%s-%d-%d", s.ChainID, s.HeightFrom, s.HeightTo)

	p.log.Infof("adding history %s", historyID)

	defer func() {
		_ = os.RemoveAll(s.ZipFilePath())
	}()

	previousHistorySegmentID, err := p.GetPreviousHistorySegmentID(s.HeightFrom)
	if err != nil {
		if !errors.Is(err, ErrSegmentNotFound) {
			return fmt.Errorf("failed to get previous history segment id:%w", err)
		}
	}

	metaData := segment.MetaData{
		Base:                     s.Base,
		PreviousHistorySegmentID: previousHistorySegmentID,
	}

	contentID, err := p.addHistorySegment(ctx, s.ZipFilePath(), metaData)
	if err != nil {
		return fmt.Errorf("failed to add file:%w", err)
	}

	p.log.Info("finished adding history to network history store",
		logging.String("history segment id", contentID.String()),
		logging.String("chain id", s.ChainID),
		logging.Int64("from height", s.HeightFrom),
		logging.Int64("to height", s.HeightTo),
		logging.String("previous history segment id", previousHistorySegmentID),
	)

	p.log.Debug("AddSnapshotData: removing old history segments")

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

func (p *Store) GetHighestBlockHeightEntry() (segment.Full, error) {
	entry, err := p.index.GetHighestBlockHeightEntry()
	if err != nil {
		if errors.Is(err, ErrIndexEntryNotFound) {
			return segment.Full{}, ErrSegmentNotFound
		}

		return segment.Full{}, fmt.Errorf("failed to get highest block height entry from index:%w", err)
	}

	return entry, nil
}

func (p *Store) ListAllIndexEntriesOldestFirst() (segment.Segments[segment.Full], error) {
	return p.index.ListAllEntriesOldestFirst()
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

func (p *Store) GetPreviousHistorySegmentID(fromHeight int64) (string, error) {
	var err error
	var previousHistorySegment segment.Full
	if fromHeight > 0 {
		height := fromHeight - 1
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

func (p *Store) addHistorySegment(ctx context.Context, zipFilePath string, metadata segment.MetaData) (cid.Cid, error) {
	newZipFile, err := p.rewriteZipWithMetadata(zipFilePath, metadata)
	defer os.Remove(newZipFile)
	if err != nil {
		return cid.Cid{}, fmt.Errorf("rewriting zip to include metadata:%w", err)
	}

	contentID, err := p.addFileToIpfs(ctx, newZipFile)
	if err != nil {
		return cid.Cid{}, fmt.Errorf("failed to add history segement %s to ipfs:%w", zipFilePath, err)
	}

	if err = p.index.Add(segment.Full{
		MetaData:         metadata,
		HistorySegmentID: contentID.String(),
	}); err != nil {
		return cid.Cid{}, fmt.Errorf("failed to update meta data store:%w", err)
	}
	return contentID, nil
}

func (p *Store) rewriteZipWithMetadata(oldZip string, metadata segment.MetaData) (string, error) {
	// Create a temporary zip file for including the metadata JSON file
	tmpfile, err := ioutil.TempFile("", metadata.ZipFileName())
	if err != nil {
		return "", fmt.Errorf("failed add history segment; unable to create temp file:%w", err)
	}

	defer tmpfile.Close()

	zipWriter := zip.NewWriter(tmpfile)
	defer zipWriter.Close()

	metaDataBytes, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal meta data:%w", err)
	}

	metaDataWriter, err := zipWriter.Create("metadata.json")
	if err != nil {
		return "", fmt.Errorf("failed to create metadata.json:%w", err)
	}

	_, err = metaDataWriter.Write(metaDataBytes)
	if err != nil {
		return "", fmt.Errorf("failed to write metadata.json:%w", err)
	}

	zipReader, err := zip.OpenReader(oldZip)
	if err != nil {
		return "", fmt.Errorf("failed to open zip file:%w", err)
	}

	// Copy the contents of the existing zip file to the new zip file
	for _, f := range zipReader.File {
		fr, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("error reading reading file from zip archive: %w", err)
		}
		defer fr.Close()

		// Create a new file header based on the existing file header and write it to the new zip file
		fw, err := zipWriter.CreateHeader(&f.FileHeader)
		if err != nil {
			return "", fmt.Errorf("error creating file header: %w", err)
		}

		// Copy the contents of the existing file to the new file
		_, err = io.Copy(fw, fr)
		if err != nil {
			return "", fmt.Errorf("error copying data from zip file: %w", err)
		}
	}

	return tmpfile.Name(), nil
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

func (p *Store) GetSegmentForHeight(toHeight int64) (segment.Full, error) {
	return p.index.Get(toHeight)
}

func (p *Store) GetHistorySegmentReader(ctx context.Context, historySegmentID string) (io.ReadSeekCloser, int64, error) {
	ipfsCid, err := cid.Parse(historySegmentID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse history segment id:%w", err)
	}

	ipfsFile, err := p.ipfsAPI.Unixfs().Get(ctx, path.IpfsPath(ipfsCid))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get ipfs file:%w", err)
	}

	fileSize, err := ipfsFile.Size()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get ipfs file size:%w", err)
	}

	return files.ToFile(ipfsFile), fileSize, nil
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

func (p *Store) removeOldHistorySegments(ctx context.Context) ([]segment.Full, error) {
	latestSegment, err := p.index.GetHighestBlockHeightEntry()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest segment:%w", err)
	}

	entries, err := p.index.ListAllEntriesOldestFirst()
	if err != nil {
		return nil, fmt.Errorf("failed to list all entries:%w", err)
	}

	var removedSegments []segment.Full
	for _, segment := range entries {
		if segment.HeightTo < (latestSegment.HeightTo - p.cfg.HistoryRetentionBlockSpan) {
			err = p.unpinSegment(ctx, segment)
			if err != nil {
				return nil, fmt.Errorf("failed to unpin segment:%w", err)
			}

			err = p.index.Remove(segment)
			if err != nil {
				return nil, fmt.Errorf("failed to remove segment from index: %w", err)
			}

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

func (p *Store) FetchHistorySegment(ctx context.Context, historySegmentID string) (segment.Full, error) {
	// We don't know what the filename is yet as that gets lost in IPFS - so we just use a generic name
	// until we peek at the metadata.json file inside to figure out the proper name and rename it.

	historySegment := filepath.Join(p.stagingDir, "segment.zip")

	err := os.RemoveAll(historySegment)
	if err != nil {
		return segment.Full{}, fmt.Errorf("failed to remove existing history segment zip: %w", err)
	}

	contentID, err := cid.Parse(historySegmentID)
	if err != nil {
		return segment.Full{}, fmt.Errorf("failed to parse snapshotId into CID:%w", err)
	}

	rootNodeFile, err := p.ipfsAPI.Unixfs().Get(ctx, path.IpfsPath(contentID))
	if err != nil {
		connInfo, swarmError := p.ipfsAPI.Swarm().Peers(ctx)
		if swarmError != nil {
			return segment.Full{}, fmt.Errorf("failed to get peers: %w", err)
		}

		peerAddrs := ""
		for _, peer := range connInfo {
			peerAddrs += fmt.Sprintf(",%s", peer.Address())
		}

		return segment.Full{}, fmt.Errorf("could not get file with CID, connected peer addresses %s: %w", peerAddrs, err)
	}

	err = files.WriteTo(rootNodeFile, historySegment)
	if err != nil {
		// check if the file exists and if so, remove it
		_, statErr := os.Stat(historySegment)
		if statErr == nil {
			remErr := os.Remove(historySegment)
			if remErr != nil {
				return segment.Full{}, fmt.Errorf("could not write out the fetched history segment: %w, and could not remove existing history segment: %v", err, remErr)
			}
		}

		return segment.Full{}, fmt.Errorf("could not write out the fetched history segment: %w", err)
	}

	zipReader, err := zip.OpenReader(historySegment)
	if err != nil {
		return segment.Full{}, fmt.Errorf("failed to open history segment: %w", err)
	}
	defer func() { _ = zipReader.Close() }()

	metaFile, err := zipReader.Open(segmentMetaDataFile)
	if err != nil {
		return segment.Full{}, fmt.Errorf("failed to open history segment metadata file: %w", err)
	}

	metaBytes, err := io.ReadAll(metaFile)
	if err != nil {
		return segment.Full{}, fmt.Errorf("failed to read index entry:%w", err)
	}

	var metaData segment.MetaData
	if err = json.Unmarshal(metaBytes, &metaData); err != nil {
		return segment.Full{}, fmt.Errorf("failed to unmarshal index entry:%w", err)
	}

	renamedSegmentPath := filepath.Join(p.stagingDir, metaData.ZipFileName())
	err = os.Rename(historySegment, renamedSegmentPath)
	if err != nil {
		return segment.Full{}, fmt.Errorf("failed to rename history segment: %w", err)
	}

	indexEntry := segment.Full{
		MetaData:         metaData,
		HistorySegmentID: historySegmentID,
	}

	err = p.ipfsAPI.Pin().Add(ctx, path.IpfsPath(contentID))
	if err != nil {
		return segment.Full{}, fmt.Errorf("failed to pin fetched segment: %w", err)
	}

	if err = p.index.Add(indexEntry); err != nil {
		return segment.Full{}, fmt.Errorf("failed to add index entry:%w", err)
	}

	return indexEntry, nil
}

func (p *Store) StagedSegment(s segment.Full) (segment.Staged, error) {
	ss := segment.Staged{
		Full:      s,
		Directory: p.stagingDir,
	}
	if _, err := os.Stat(ss.ZipFilePath()); err != nil {
		return segment.Staged{}, fmt.Errorf("segment %v not fetched into staging area:%w", s, err)
	}
	return ss, nil
}

func (p *Store) StagedContiguousHistory(chunk segment.ContiguousHistory[segment.Full]) (segment.ContiguousHistory[segment.Staged], error) {
	staged := segment.ContiguousHistory[segment.Staged]{}

	for _, s := range chunk.Segments {
		ss, err := p.StagedSegment(s)
		if err != nil {
			return segment.ContiguousHistory[segment.Staged]{}, err
		}
		if ok := staged.Add(ss); !ok {
			return segment.ContiguousHistory[segment.Staged]{}, fmt.Errorf("failed to build staged chunk; input chunk not contiguous")
		}
	}

	return staged, nil
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

func generateSwarmKeyFile(swarmKeySeed string, repoPath string) (string, error) {
	file, err := os.Create(filepath.Join(repoPath, "swarm.key"))
	defer func() { _ = file.Close() }()
	if err != nil {
		return "", fmt.Errorf("failed to create swarm key file:%w", err)
	}

	key := make([]byte, 32)

	copy(key, swarmKeySeed)
	hx := hex.EncodeToString(key)

	swarmKey := fmt.Sprintf("/key/swarm/psk/1.0.0/\n/base16/\n%s", hx)
	_, err = io.WriteString(file, swarmKey)

	if err != nil {
		return "", fmt.Errorf("failed to write to file:%w", err)
	}

	return swarmKey, nil
}

func createNode(ctx context.Context, log *logging.Logger, repoPath string, maxMemoryPercent uint8) (*core.IpfsNode, repo.Repo, error) {
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

	err = setLibP2PResourceManagerLimits(repo, maxMemoryPercent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to set libp2p resource manager limits:%w", err)
	}

	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create new node:%w", err)
	}

	printSwarmAddrs(node, log)

	// Attach the Core API to the constructed node
	return node, repo, nil
}

// The LibP2P Resource manager protects the IPFS node from malicious and non-malicious attacks, the limits used to enforce
// these protections are based on the max memory and max file descriptor limits set in the swarms resource manager config.
// This method overrides the defaults and sets limits that we consider sensible in the context of a data-node.
func setLibP2PResourceManagerLimits(repo repo.Repo, maxMemoryPercent uint8) error {
	cfg, err := repo.Config()
	if err != nil {
		return fmt.Errorf("failed to get repo config:%w", err)
	}

	// Use max memory percent if set, otherwise use libP2P defaults
	if maxMemoryPercent > 0 {
		totalMem, err := memory.TotalMemory()
		if err != nil {
			return fmt.Errorf("failed to get total memory: %w", err)
		}

		// Set the maximum to a quarter of the data-nodes max memory
		maxMemoryString := humanize.Bytes(uint64(float64(totalMem) * (float64(maxMemoryPercent) / (4 * 100))))
		cfg.Swarm.ResourceMgr.MaxMemory = config.NewOptionalString(maxMemoryString)
	}

	// Set the maximum to a quarter of the systems available file descriptors
	maxFileDescriptors := int64(fd.GetNumFDs()) / 4
	fdBytes, err := json.Marshal(&maxFileDescriptors)
	if err != nil {
		return fmt.Errorf("failed to marshal max file descriptors:%w", err)
	}

	fdOptionalInteger := config.OptionalInteger{}
	err = fdOptionalInteger.UnmarshalJSON(fdBytes)
	if err != nil {
		return fmt.Errorf("failed to unmarshal max file descriptors:%w", err)
	}

	cfg.Swarm.ResourceMgr.MaxFileDescriptors = &fdOptionalInteger

	return nil
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
	cfg *config.Config, swarmKeySeed string, maxMemoryPercent uint8,
) (*core.IpfsNode, repo.Repo, string, error) {
	// Only inits the repo if it does not already exist
	err := fsrepo.Init(repoPath, cfg)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to initialise ipfs configuration:%w", err)
	}

	// Update to take account of any new bootstrap nodes
	err = updateRepoConfig(repoPath, cfg)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to update ipfs configuration:%w", err)
	}

	swarmKey, err := generateSwarmKeyFile(swarmKeySeed, repoPath)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to generate swarm key file:%w", err)
	}

	node, repo, err := createNode(ctx, log, repoPath, maxMemoryPercent)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create node: %w", err)
	}

	return node, repo, swarmKey, nil
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

func (p *Store) unpinSegment(ctx context.Context, segment segment.Full) error {
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
