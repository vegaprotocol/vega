package networkhistory

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"code.vegaprotocol.io/vega/datanode/networkhistory/fsutil"
	"github.com/multiformats/go-multiaddr"

	"code.vegaprotocol.io/vega/datanode/networkhistory/snapshot"
	"code.vegaprotocol.io/vega/datanode/networkhistory/store"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/logging"
)

type Segment interface {
	GetFromHeight() int64
	GetToHeight() int64
	GetHistorySegmentId() string
	GetPreviousHistorySegmentId() string
}

type Service struct {
	log      *logging.Logger
	connPool *pgxpool.Pool

	snapshotService *snapshot.Service
	store           *store.Store

	chainID string

	snapshotsCopyFromDir string
	snapshotsCopyToDir   string

	datanodeGrpcAPIPort int

	publishLock sync.Mutex
}

func New(ctx context.Context, log *logging.Logger, cfg Config, networkHistoryHome string, connPool *pgxpool.Pool, connConfig sqlstore.ConnectionConfig,
	chainID string,
	snapshotService *snapshot.Service, datanodeGrpcAPIPort int,
	snapshotsCopyFromDir, snapshotsCopyToDir string,
) (*Service, error) {
	storeLog := log.Named("store")
	storeLog.SetLevel(cfg.Level.Get())

	networkHistoryStore, err := store.New(ctx, storeLog, chainID, cfg.Store, networkHistoryHome, bool(cfg.WipeOnStartup))
	if err != nil {
		return nil, fmt.Errorf("failed to create network history store:%w", err)
	}

	return NewWithStore(ctx, log, chainID, cfg, connPool, snapshotService, networkHistoryStore, datanodeGrpcAPIPort, snapshotsCopyFromDir, snapshotsCopyToDir)
}

func NewWithStore(ctx context.Context, log *logging.Logger, chainID string, cfg Config, connPool *pgxpool.Pool,
	snapshotService *snapshot.Service,
	networkHistoryStore *store.Store, datanodeGrpcAPIPort int,
	snapshotsCopyFromDir, snapshotsCopyToDir string,
) (*Service, error) {
	s := &Service{
		log:                  log,
		connPool:             connPool,
		snapshotService:      snapshotService,
		store:                networkHistoryStore,
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

func (d *Service) GetHighestBlockHeightHistorySegment() (Segment, error) {
	return d.store.GetHighestBlockHeightEntry()
}

func (d *Service) ListAllHistorySegments() ([]Segment, error) {
	indexEntries, err := d.store.ListAllIndexEntriesOldestFirst()
	if err != nil {
		return nil, fmt.Errorf("failed to list all index entries")
	}

	result := make([]Segment, 0, len(indexEntries))
	for _, indexEntry := range indexEntries {
		result = append(result, indexEntry)
	}

	return result, nil
}

func (d *Service) FetchHistorySegment(ctx context.Context, historySegmentID string) (Segment, error) {
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

func (d *Service) LoadNetworkHistoryIntoDatanode(ctx context.Context, contiguousHistory ContiguousHistory,
	connConfig sqlstore.ConnectionConfig, withIndexesAndOrderTriggers bool,
) (snapshot.LoadResult, error) {
	return d.LoadNetworkHistoryIntoDatanodeWithLog(ctx, d.log, contiguousHistory, connConfig, withIndexesAndOrderTriggers)
}

func (d *Service) LoadNetworkHistoryIntoDatanodeWithLog(ctx context.Context, loadLog snapshot.LoadLog, contiguousHistory ContiguousHistory,
	connConfig sqlstore.ConnectionConfig, withIndexesAndOrderTriggers bool,
) (snapshot.LoadResult, error) {
	defer func() { _ = fsutil.RemoveAllFromDirectoryIfExists(d.snapshotsCopyFromDir) }()

	datanodeBlockSpan, err := sqlstore.GetDatanodeBlockSpan(ctx, d.connPool)
	if err != nil {
		return snapshot.LoadResult{}, fmt.Errorf("failed to get data node block span: %w", err)
	}

	loadLog.Info("loading network history into the datanode", logging.Int64("fromHeight", contiguousHistory.HeightFrom),
		logging.Int64("toHeight", contiguousHistory.HeightTo), logging.Int64("currentDatanodeFromHeight", datanodeBlockSpan.FromHeight),
		logging.Int64("currentDatanodeToHeight", datanodeBlockSpan.ToHeight), logging.Bool("withIndexesAndOrderTriggers", withIndexesAndOrderTriggers))

	err = os.MkdirAll(d.snapshotsCopyFromDir, fs.ModePerm)
	if err != nil {
		return snapshot.LoadResult{}, fmt.Errorf("failed to create staging directory:%w", err)
	}

	err = fsutil.RemoveAllFromDirectoryIfExists(d.snapshotsCopyFromDir)
	if err != nil {
		return snapshot.LoadResult{}, fmt.Errorf("failed to empty staging directory:%w", err)
	}

	start := time.Now()

	currentStateSnapshot, historySnapshots, err := d.copyMoreRecentHistoryIntoDir(ctx, contiguousHistory, datanodeBlockSpan, d.snapshotsCopyFromDir)
	if err != nil {
		return snapshot.LoadResult{}, fmt.Errorf("failed to copy all available data into copy from path: %w", err)
	}

	if len(historySnapshots) == 0 {
		return snapshot.LoadResult{}, fmt.Errorf("no data available to load: %w", err)
	}

	loadResult, err := d.snapshotService.LoadSnapshotData(ctx, loadLog, currentStateSnapshot, historySnapshots, d.snapshotsCopyFromDir,
		connConfig, withIndexesAndOrderTriggers)
	if err != nil {
		return snapshot.LoadResult{}, fmt.Errorf("failed to load snapshot data:%w", err)
	}

	loadLog.Info("loaded all available data into datanode", logging.String("result", fmt.Sprintf("%+v", loadResult)),
		logging.Duration("time taken", time.Since(start)))
	return loadResult, err
}

func (d *Service) GetMostRecentHistorySegmentFromPeers(ctx context.Context,
	grpcAPIPorts []int,
) (*PeerResponse, map[string]*v2.GetMostRecentNetworkHistorySegmentResponse, error) {
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
		return nil, nil, errors.New("no active peers found")
	}

	return GetMostRecentHistorySegmentFromPeersAddresses(ctx, activePeerAddresses, d.GetSwarmKey(), grpcAPIPorts)
}

func (d *Service) GetDatanodeBlockSpan(ctx context.Context) (sqlstore.DatanodeBlockSpan, error) {
	return sqlstore.GetDatanodeBlockSpan(ctx, d.connPool)
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

// copyMoreRecentHistoryIntoDir copies all contiguous history data later than that already in the datanode into the target directory.
func (d *Service) copyMoreRecentHistoryIntoDir(ctx context.Context, contiguousHistory ContiguousHistory,
	blockSpan sqlstore.DatanodeBlockSpan, targetDir string) (snapshot.CurrentState, []snapshot.History,
	error,
) {
	var highestCurrentStateSnapshot snapshot.CurrentState
	contiguousHistorySnapshots := make([]snapshot.History, 0, len(contiguousHistory.SegmentsOldestFirst))
	for _, history := range contiguousHistory.SegmentsOldestFirst {
		if history.GetToHeight() > blockSpan.ToHeight {
			currentStateSnaphot, historySnapshot, err := d.extractSnapshotDataFromHistory(ctx, history, targetDir)
			if err != nil {
				return snapshot.CurrentState{}, nil, fmt.Errorf("failed to extract data from history:%w", err)
			}

			if currentStateSnaphot.Height > highestCurrentStateSnapshot.Height {
				highestCurrentStateSnapshot = currentStateSnaphot
			}

			contiguousHistorySnapshots = append(contiguousHistorySnapshots, historySnapshot)
		}
	}

	return highestCurrentStateSnapshot, contiguousHistorySnapshots, nil
}

func (d *Service) extractSnapshotDataFromHistory(ctx context.Context, history Segment, targetDir string) (snapshot.CurrentState, snapshot.History, error) {
	var err error
	var currentStateSnaphot snapshot.CurrentState
	var historySnapshot snapshot.History

	currentStateSnaphot, historySnapshot, err = d.store.CopySnapshotDataIntoDir(ctx, history.GetToHeight(), targetDir)
	if err != nil {
		return snapshot.CurrentState{}, snapshot.History{}, fmt.Errorf("failed to extract history segment for height: %d: %w", history.GetToHeight(), err)
	}

	return currentStateSnaphot, historySnapshot, nil
}

func (d *Service) Stop() {
	d.log.Info("stopping network history service")
	d.store.Stop()
	d.connPool.Close()
}

func KillAllConnectionsToDatabase(ctx context.Context, connConfig sqlstore.ConnectionConfig) error {
	conn, err := pgxpool.Connect(ctx, connConfig.GetConnectionString())
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close()

	killAllConnectionsQuery := fmt.Sprintf(
		`SELECT
	pg_terminate_backend(pg_stat_activity.pid)
		FROM
	pg_stat_activity
		WHERE
	pg_stat_activity.datname = '%s'
	AND pid <> pg_backend_pid();`, connConfig.Database)

	_, err = conn.Exec(ctx, killAllConnectionsQuery)
	if err != nil {
		return fmt.Errorf("failed to kill all database connection: %w", err)
	}

	return nil
}
