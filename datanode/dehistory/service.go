package dehistory

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"sync"
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"code.vegaprotocol.io/vega/datanode/dehistory/aggregation"
	"code.vegaprotocol.io/vega/datanode/dehistory/fsutil"
	"github.com/multiformats/go-multiaddr"

	"code.vegaprotocol.io/vega/datanode/dehistory/snapshot"
	"code.vegaprotocol.io/vega/datanode/dehistory/store"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/logging"
)

var ErrNoActivePeersFound = errors.New("no active peers found")

type Service struct {
	log *logging.Logger
	cfg Config

	snapshotService *snapshot.Service
	store           *store.Store
	connConfig      sqlstore.ConnectionConfig

	chainID string

	snapshotsCopyFromDir string
	snapshotsCopyToDir   string

	datanodeGrpcAPIPort int

	publishLock sync.Mutex
}

func New(ctx context.Context, log *logging.Logger, cfg Config, deHistoryHome string, connConfig sqlstore.ConnectionConfig,
	chainID string,
	snapshotService *snapshot.Service, datanodeGrpcAPIPort int,
	snapshotsCopyFromDir, snapshotsCopyToDir string,
) (*Service, error) {
	storeLog := log.Named("store")
	storeLog.SetLevel(cfg.Level.Get())

	deHistoryStore, err := store.New(ctx, storeLog, chainID, cfg.Store, deHistoryHome, bool(cfg.WipeOnStartup))
	if err != nil {
		return nil, fmt.Errorf("failed to create decentralized history store:%w", err)
	}

	return NewWithStore(ctx, log, chainID, cfg, connConfig, snapshotService, deHistoryStore, datanodeGrpcAPIPort, snapshotsCopyFromDir, snapshotsCopyToDir)
}

func NewWithStore(ctx context.Context, log *logging.Logger, chainID string, cfg Config, connConfig sqlstore.ConnectionConfig,
	snapshotService *snapshot.Service,
	deHistoryStore *store.Store, datanodeGrpcAPIPort int,
	snapshotsCopyFromDir, snapshotsCopyToDir string,
) (*Service, error) {
	s := &Service{
		log:                  log,
		cfg:                  cfg,
		snapshotService:      snapshotService,
		store:                deHistoryStore,
		connConfig:           connConfig,
		chainID:              chainID,
		snapshotsCopyFromDir: snapshotsCopyFromDir,
		snapshotsCopyToDir:   snapshotsCopyToDir,
		datanodeGrpcAPIPort:  datanodeGrpcAPIPort,
	}

	if cfg.WipeOnStartup {
		err := fsutil.RemoveAllFromDirectoryIfExists(s.snapshotsCopyFromDir)
		if err != nil {
			return nil, fmt.Errorf("failed to remove all from snapshots copy from path:%w", err)
		}

		err = fsutil.RemoveAllFromDirectoryIfExists(s.snapshotsCopyToDir)
		if err != nil {
			return nil, fmt.Errorf("failed to remove all from snapshots copy to path:%w", err)
		}
	}

	if cfg.Publish {
		var err error
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					err = s.publishSnapshots(ctx)
					if err != nil {
						s.log.Errorf("failed to add all snapshot data to store:%s", err)
					}
				}
			}
		}()
	}

	return s, nil
}

func (d *Service) CopyHistorySegmentToFile(ctx context.Context, historySegmentID string, outFile string) error {
	return d.store.CopyHistorySegmentToFile(ctx, historySegmentID, outFile)
}

func (d *Service) GetHighestBlockHeightHistorySegment() (store.SegmentIndexEntry, error) {
	return d.store.GetHighestBlockHeightEntry()
}

func (d *Service) ListAllHistorySegments() ([]store.SegmentIndexEntry, error) {
	return d.store.ListAllHistorySegmentsOldestFirst()
}

func (d *Service) FetchHistorySegment(ctx context.Context, historySegmentID string) (store.SegmentIndexEntry, error) {
	return d.store.FetchHistorySegment(ctx, historySegmentID)
}

func (d *Service) CreateAndPublishSegment(ctx context.Context, chainID string, toHeight int64) error {
	_, err := d.snapshotService.CreateSnapshot(ctx, chainID, toHeight)
	if err != nil {
		if !errors.Is(err, snapshot.ErrSnapshotExists) {
			return fmt.Errorf("failed to create snapshot: %w", err)
		}
	}

	if err = d.publishSnapshots(ctx); err != nil {
		return fmt.Errorf("failed to publish snapshots: %w", err)
	}

	return nil
}

func (d *Service) GetActivePeerAddresses() []string {
	ip4Protocol := multiaddr.ProtocolWithName("ip4")
	ip6Protocol := multiaddr.ProtocolWithName("ip6")
	var activePeerIPAddresses []string

	activePeerIPAddresses = nil
	peerAddresses := d.store.GetPeerAddrs()

	for _, addr := range peerAddresses {
		ipAddr, err := addr.ValueForProtocol(ip4Protocol.Code)
		if err == nil {
			activePeerIPAddresses = append(activePeerIPAddresses, ipAddr)
		}

		ipAddr, err = addr.ValueForProtocol(ip6Protocol.Code)
		if err == nil {
			activePeerIPAddresses = append(activePeerIPAddresses, ipAddr)
		}
	}

	return activePeerIPAddresses
}

func (d *Service) GetSwarmKey() string {
	return d.store.GetSwarmKey()
}

func (d *Service) LoadAllAvailableHistoryIntoDatanode(ctx context.Context, sqlFs fs.FS) (snapshot.LoadResult, error) {
	defer func() { _ = fsutil.RemoveAllFromDirectoryIfExists(d.snapshotsCopyFromDir) }()

	err := os.MkdirAll(d.snapshotsCopyFromDir, fs.ModePerm)
	if err != nil {
		return snapshot.LoadResult{}, fmt.Errorf("failed to create staging directory:%w", err)
	}

	err = fsutil.RemoveAllFromDirectoryIfExists(d.snapshotsCopyFromDir)
	if err != nil {
		return snapshot.LoadResult{}, fmt.Errorf("failed to empty staging directory:%w", err)
	}

	start := time.Now()

	currentStateSnapshot, contiguousHistory, err := d.copyAllAvailableHistoryIntoDir(ctx, d.snapshotsCopyFromDir)
	if err != nil {
		return snapshot.LoadResult{}, fmt.Errorf("failed to copy all available data into copy from path: %w", err)
	}

	if len(contiguousHistory) == 0 {
		return snapshot.LoadResult{}, fmt.Errorf("no data available to load: %w", err)
	}

	loadResult, err := d.snapshotService.LoadAllSnapshotData(ctx, currentStateSnapshot, contiguousHistory, d.snapshotsCopyFromDir)
	if err != nil {
		return snapshot.LoadResult{}, fmt.Errorf("failed to load snapshot data:%w", err)
	}

	d.log.Info("loaded all available data into datanode", logging.String("result", fmt.Sprintf("%+v", loadResult)),
		logging.Duration("time taken", time.Since(start)))
	return loadResult, err
}

func (d *Service) GetMostRecentHistorySegmentFromPeers(ctx context.Context,
	grpcAPIPorts []int,
) (*PeerResponse, map[string]*v2.GetMostRecentDeHistorySegmentResponse, error) {
	var activePeerAddresses []string
	// Time for connections to be established
	time.Sleep(5 * time.Second)
	for retries := 0; retries < 5; retries++ {
		activePeerAddresses = d.GetActivePeerAddresses()
		if len(activePeerAddresses) == 0 {
			time.Sleep(5 * time.Second)
		}
	}

	if len(activePeerAddresses) == 0 {
		return nil, nil, ErrNoActivePeersFound
	}

	return GetMostRecentHistorySegmentFromPeersAddresses(ctx, activePeerAddresses, d.GetSwarmKey(), grpcAPIPorts)
}

func (d *Service) publishSnapshots(ctx context.Context) error {
	d.publishLock.Lock()
	defer d.publishLock.Unlock()

	_, snapshots, err := snapshot.GetCurrentStateSnapshots(d.snapshotsCopyToDir)
	if err != nil {
		return fmt.Errorf("failed to get current state snapshots:%w", err)
	}

	snapshotsOldestFirst := make([]snapshot.CurrentState, 0, len(snapshots))
	for _, currentStateSnapshot := range snapshots {
		snapshotsOldestFirst = append(snapshotsOldestFirst, currentStateSnapshot)
	}

	sort.Slice(snapshotsOldestFirst, func(i, j int) bool {
		return snapshotsOldestFirst[i].Height < snapshotsOldestFirst[j].Height
	})

	_, histories, err := snapshot.GetHistorySnapshots(d.snapshotsCopyToDir)
	if err != nil {
		return fmt.Errorf("failed to get history snapshots:%w", err)
	}

	heightToHistory := map[int64]snapshot.History{}
	for _, history := range histories {
		heightToHistory[history.HeightTo] = history
	}

	for _, currentState := range snapshotsOldestFirst {
		history, ok := heightToHistory[currentState.Height]
		if !ok {
			return fmt.Errorf("failed to find history for current state snapshot:%w", err)
		}

		err = d.store.AddSnapshotData(ctx, history, currentState, d.snapshotsCopyToDir)
		if err != nil {
			return fmt.Errorf("failed to publish snapshot %s:%w", currentState, err)
		}
	}

	return nil
}

// copyAllAvailableHistoryIntoDir copy all contiguous history data, including data already loaded into the datanode to the target dir.
func (d *Service) copyAllAvailableHistoryIntoDir(ctx context.Context, targetDir string) (snapshot.CurrentState, []snapshot.History,
	error,
) {
	contiguousHistory, err := d.GetContiguousHistory()
	if err != nil {
		return snapshot.CurrentState{}, nil, fmt.Errorf("failed to get contiguous history data")
	}

	if len(contiguousHistory) == 0 {
		return snapshot.CurrentState{}, nil, fmt.Errorf("no contiguous history data available")
	}

	var highestCurrentStateSnapshot snapshot.CurrentState
	contiguousHistorySnapshots := make([]snapshot.History, 0, len(contiguousHistory))
	for _, history := range contiguousHistory {
		currentStateSnaphot, historySnapshot, err := d.extractSnapshotDataFromHistory(ctx, history, targetDir)
		if err != nil {
			return snapshot.CurrentState{}, nil, fmt.Errorf("failed to extract data from history:%w", err)
		}

		if currentStateSnaphot.Height > highestCurrentStateSnapshot.Height {
			highestCurrentStateSnapshot = currentStateSnaphot
		}

		contiguousHistorySnapshots = append(contiguousHistorySnapshots, historySnapshot)
	}

	return highestCurrentStateSnapshot, contiguousHistorySnapshots, nil
}

// GetContiguousHistory returns all available contiguous (no gaps) history from the current datanode height, or if
// the datanode has no data it will return the contiguous history from the highest decentralized history segment.
func (d *Service) GetContiguousHistory() ([]aggregation.AggregatedHistorySegment, error) {
	allHistorySegments, err := d.store.ListAllHistorySegmentsOldestFirst()
	if err != nil {
		return nil, fmt.Errorf("failed to get all history segments:%w", err)
	}

	contiguousHistory := aggregation.GetHighestContiguousHistoryFromSegmentIndexEntry(allHistorySegments)

	return contiguousHistory, nil
}

func (d *Service) extractSnapshotDataFromHistory(ctx context.Context, history aggregation.AggregatedHistorySegment, targetDir string) (snapshot.CurrentState, snapshot.History, error) {
	var err error
	var currentStateSnaphot snapshot.CurrentState
	var historySnapshot snapshot.History

	currentStateSnaphot, historySnapshot, err = d.store.CopySnapshotDataIntoDir(ctx, history.HeightTo, targetDir)
	if err != nil {
		return snapshot.CurrentState{}, snapshot.History{}, fmt.Errorf("failed to extract history segment for height: %d: %w", history.HeightTo, err)
	}

	return currentStateSnaphot, historySnapshot, nil
}
