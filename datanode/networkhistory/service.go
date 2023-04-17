package networkhistory

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/multiformats/go-multiaddr"

	"github.com/jackc/pgx/v4/pgxpool"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"code.vegaprotocol.io/vega/datanode/networkhistory/fsutil"
	"code.vegaprotocol.io/vega/datanode/networkhistory/segment"
	"code.vegaprotocol.io/vega/datanode/networkhistory/snapshot"
	"code.vegaprotocol.io/vega/datanode/networkhistory/store"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/logging"
)

type Service struct {
	cfg Config

	log      *logging.Logger
	connPool *pgxpool.Pool

	snapshotService *snapshot.Service
	store           *store.Store

	chainID string

	snapshotsCopyToPath string

	datanodeGrpcAPIPort int

	publishLock sync.Mutex
}

func New(ctx context.Context, log *logging.Logger, cfg Config, networkHistoryHome string, connPool *pgxpool.Pool,
	chainID string,
	snapshotService *snapshot.Service, datanodeGrpcAPIPort int,
	snapshotsCopyFromDir, snapshotsCopyToDir string, maxMemoryPercent uint8,
) (*Service, error) {
	storeLog := log.Named("store")
	storeLog.SetLevel(cfg.Level.Get())

	networkHistoryStore, err := store.New(ctx, storeLog, chainID, cfg.Store, networkHistoryHome,
		bool(cfg.WipeOnStartup), maxMemoryPercent)
	if err != nil {
		return nil, fmt.Errorf("failed to create network history store:%w", err)
	}

	return NewWithStore(ctx, log, chainID, cfg, connPool, snapshotService, networkHistoryStore, datanodeGrpcAPIPort,
		snapshotsCopyToDir)
}

func NewWithStore(ctx context.Context, log *logging.Logger, chainID string, cfg Config, connPool *pgxpool.Pool,
	snapshotService *snapshot.Service,
	networkHistoryStore *store.Store, datanodeGrpcAPIPort int,
	snapshotsCopyToPath string,
) (*Service, error) {
	s := &Service{
		cfg:                 cfg,
		log:                 log,
		connPool:            connPool,
		snapshotService:     snapshotService,
		store:               networkHistoryStore,
		chainID:             chainID,
		snapshotsCopyToPath: snapshotsCopyToPath,
		datanodeGrpcAPIPort: datanodeGrpcAPIPort,
	}

	if cfg.WipeOnStartup {
		err := fsutil.RemoveAllFromDirectoryIfExists(s.snapshotsCopyToPath)
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

func (d *Service) RollbackToHeight(ctx context.Context, log snapshot.LoadLog, height int64) error {
	datanodeBlockSpan, err := sqlstore.GetDatanodeBlockSpan(ctx, d.connPool)
	if err != nil {
		return fmt.Errorf("failed to get data node block span: %w", err)
	}

	if height < datanodeBlockSpan.FromHeight || height >= datanodeBlockSpan.ToHeight {
		return fmt.Errorf("rollback to height, %d, is not within the datanodes current block span, %d to %d",
			height, datanodeBlockSpan.FromHeight, datanodeBlockSpan.ToHeight)
	}

	rollbackToSegment, err := d.store.GetSegmentForHeight(height)
	if err != nil {
		return fmt.Errorf("failed to get history segment for height %d: %w", height, err)
	}

	stagedSegment, err := d.store.StagedSegment(rollbackToSegment)
	if err != nil {
		return fmt.Errorf("failed to get staged segment for height %d: %w", height, err)
	}

	err = d.snapshotService.RollbackToSegment(ctx, log, stagedSegment)

	if err != nil {
		return fmt.Errorf("failed to rollback to segment: %w", err)
	}

	log.Infof("finished rolling back to height %d", height)

	return nil
}

func (d *Service) GetHistorySegmentReader(ctx context.Context, historySegmentID string) (io.ReadSeekCloser, int64, error) {
	return d.store.GetHistorySegmentReader(ctx, historySegmentID)
}

func (d *Service) CopyHistorySegmentToFile(ctx context.Context, historySegmentID string, outFile string) error {
	return d.store.CopyHistorySegmentToFile(ctx, historySegmentID, outFile)
}

func (d *Service) GetHighestBlockHeightHistorySegment() (segment.Full, error) {
	return d.store.GetHighestBlockHeightEntry()
}

func (d *Service) ListAllHistorySegments() (segment.Segments[segment.Full], error) {
	return d.store.ListAllIndexEntriesOldestFirst()
}

func (d *Service) FetchHistorySegment(ctx context.Context, historySegmentID string) (segment.Full, error) {
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

func (d *Service) GetBootstrapPeers() []string {
	return d.cfg.Store.BootstrapPeers
}

func (d *Service) GetSwarmKey() string {
	return d.store.GetSwarmKey()
}

func (d *Service) GetIpfsAddress() (string, error) {
	node, err := d.store.GetLocalNode()
	if err != nil {
		return "", fmt.Errorf("failed to load node: %w", err)
	}

	ipfsAddress, err := node.IpfsAddress()
	if err != nil {
		return "", fmt.Errorf("failed to get ipfs address: %w", err)
	}

	return ipfsAddress.String(), nil
}

func (d *Service) GetConnectedPeerAddresses() ([]string, error) {
	connectedPeers := d.store.GetConnectedPeers()

	addr := make([]string, 0, len(connectedPeers))
	for _, peer := range connectedPeers {
		ipfsAddress, err := peer.Remote.IpfsAddress()
		if err != nil {
			return nil, fmt.Errorf("failed to get ipfs address of remote peer: %w", err)
		}
		addr = append(addr, ipfsAddress.String())
	}

	return addr, nil
}

func (d *Service) GetActivePeerIPAddresses() []string {
	ip4Protocol := multiaddr.ProtocolWithName("ip4")
	ip6Protocol := multiaddr.ProtocolWithName("ip6")
	var activePeerIPAddresses []string

	activePeerIPAddresses = nil
	connectedPeers := d.store.GetConnectedPeers()

	for _, addr := range connectedPeers {
		ipAddr, err := addr.Remote.Addr.ValueForProtocol(ip4Protocol.Code)
		if err == nil {
			activePeerIPAddresses = append(activePeerIPAddresses, ipAddr)
		}

		ipAddr, err = addr.Remote.Addr.ValueForProtocol(ip6Protocol.Code)
		if err == nil {
			activePeerIPAddresses = append(activePeerIPAddresses, ipAddr)
		}
	}

	return activePeerIPAddresses
}

func (d *Service) GetSwarmKeySeed() string {
	return d.store.GetSwarmKeySeed()
}

func (d *Service) LoadNetworkHistoryIntoDatanode(ctx context.Context, chunk segment.ContiguousHistory[segment.Full],
	connConfig sqlstore.ConnectionConfig, withIndexesAndOrderTriggers, verbose bool,
) (snapshot.LoadResult, error) {
	return d.LoadNetworkHistoryIntoDatanodeWithLog(ctx, d.log, chunk, connConfig, withIndexesAndOrderTriggers, verbose)
}

func (d *Service) LoadNetworkHistoryIntoDatanodeWithLog(ctx context.Context, log snapshot.LoadLog, chunk segment.ContiguousHistory[segment.Full],
	connConfig sqlstore.ConnectionConfig, withIndexesAndOrderTriggers, verbose bool,
) (snapshot.LoadResult, error) {
	datanodeBlockSpan, err := sqlstore.GetDatanodeBlockSpan(ctx, d.connPool)
	if err != nil {
		return snapshot.LoadResult{}, fmt.Errorf("failed to get data node block span: %w", err)
	}

	log.Info("loading network history into the datanode", logging.Int64("fromHeight", chunk.HeightFrom),
		logging.Int64("toHeight", chunk.HeightFrom), logging.Int64("currentDatanodeFromHeight", datanodeBlockSpan.FromHeight),
		logging.Int64("currentDatanodeToHeight", datanodeBlockSpan.ToHeight), logging.Bool("withIndexesAndOrderTriggers", withIndexesAndOrderTriggers))

	start := time.Now()

	chunkToLoad := chunk.Slice(datanodeBlockSpan.ToHeight+1, chunk.HeightTo)
	stagedChunk, err := d.store.StagedContiguousHistory(chunkToLoad)
	if err != nil {
		return snapshot.LoadResult{}, fmt.Errorf("failed to load history:%w", err)
	}

	loadResult, err := d.snapshotService.LoadSnapshotData(ctx, log, stagedChunk, connConfig, withIndexesAndOrderTriggers, verbose)
	if err != nil {
		return snapshot.LoadResult{}, fmt.Errorf("failed to load snapshot data:%w", err)
	}

	log.Info("loaded all available data into datanode", logging.String("result", fmt.Sprintf("%+v", loadResult)),
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
		activePeerAddresses = d.GetActivePeerIPAddresses()
		if len(activePeerAddresses) == 0 {
			time.Sleep(5 * time.Second)
		}
	}

	if len(activePeerAddresses) == 0 {
		return nil, nil, errors.New("no active peers found")
	}

	return GetMostRecentHistorySegmentFromPeersAddresses(ctx, activePeerAddresses, d.GetSwarmKeySeed(), grpcAPIPorts)
}

func (d *Service) GetDatanodeBlockSpan(ctx context.Context) (sqlstore.DatanodeBlockSpan, error) {
	return sqlstore.GetDatanodeBlockSpan(ctx, d.connPool)
}

func (d *Service) publishSnapshots(ctx context.Context) error {
	d.publishLock.Lock()
	defer d.publishLock.Unlock()

	segments, err := d.snapshotService.GetUnpublishedSnapshots()
	if err != nil {
		return fmt.Errorf("failed to list snapshots:%w", err)
	}

	sort.Slice(segments, func(i, j int) bool {
		return segments[i].HeightTo < segments[j].HeightTo
	})

	for _, segment := range segments {
		err = d.store.AddSnapshotData(ctx, segment)
		if err != nil {
			return fmt.Errorf("failed to publish snapshot %s:%w", segment, err)
		}
	}

	return nil
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
