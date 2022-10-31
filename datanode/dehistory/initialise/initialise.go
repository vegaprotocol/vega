package initialise

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sort"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/datanode/service"

	"code.vegaprotocol.io/vega/datanode/dehistory/snapshot"
	"code.vegaprotocol.io/vega/datanode/dehistory/store"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/jackc/pgx/v4/pgxpool"
	"google.golang.org/grpc"
)

var (
	ErrFailedToGetSegment = errors.New("no history segment found")
	ErrChainNotFound      = errors.New("no chain found")
)

type deHistoryService interface {
	GetActivePeerAddresses() []string
	FetchHistorySegment(ctx context.Context, historySegmentID string) (store.SegmentIndexEntry, error)
	LoadAllAvailableHistoryIntoDatanode(ctx context.Context) (loadedFrom int64, loadedTo int64, err error)
}

func DatanodeFromDeHistory(parentCtx context.Context, cfg Config, log *logging.Logger,
	deHistoryService deHistoryService, grpcAPIPort int,
) (err error) {
	ctx, ctxCancelFn := context.WithTimeout(parentCtx, cfg.TimeOut.Duration)
	defer ctxCancelFn()

	var activePeerAddresses []string
	// Time for connections to be established
	time.Sleep(5 * time.Second)
	for retries := 0; retries < 5; retries++ {
		activePeerAddresses = deHistoryService.GetActivePeerAddresses()
		if len(activePeerAddresses) == 0 {
			time.Sleep(5 * time.Second)
		}
	}

	if len(activePeerAddresses) == 0 {
		return fmt.Errorf("failed to find any active peer addresses")
	}

	mostRecentHistorySegmentFromPeers, _, err := GetMostRecentHistorySegmentFromPeers(ctx, activePeerAddresses, grpcAPIPort)
	if err != nil {
		return fmt.Errorf("failed to get most recent history segment from peers:%w", err)
	}

	log.Infof("got most recent history segment:%s", mostRecentHistorySegmentFromPeers)

	log.Infof("fetching history using as the first segment:{%s} and minimum block count of %d", mostRecentHistorySegmentFromPeers, cfg.MinimumBlockCount)

	blocksFetched, err := FetchHistoryBlocks(ctx, log.Infof, mostRecentHistorySegmentFromPeers.HistorySegmentId,
		func(ctx context.Context, historySegmentID string) (FetchResult, error) {
			segment, err := deHistoryService.FetchHistorySegment(ctx, historySegmentID)
			if err != nil {
				return FetchResult{}, err
			}
			return FromSegmentIndexEntry(segment), nil
		}, cfg.MinimumBlockCount)

	if blocksFetched == 0 {
		return fmt.Errorf("failed to get any blocks from decentralised history")
	}

	log.Infof("fetched %d blocks from decentralised history", blocksFetched)

	log.Infof("loading history into the datanode")
	from, to, err := deHistoryService.LoadAllAvailableHistoryIntoDatanode(ctx)
	if err != nil {
		return fmt.Errorf("failed to load history into the datanode%w", err)
	}
	log.Infof("loaded history from height %d to %d into the datanode", from, to)

	return nil
}

func GetMostRecentHistorySegmentFromPeers(ctx context.Context, peerAddresses []string,
	grpcAPIPort int,
) (*v2.HistorySegment, map[string]*v2.HistorySegment, error) {
	const maxPeersToContact = 10

	if len(peerAddresses) > maxPeersToContact {
		peerAddresses = peerAddresses[:maxPeersToContact]
	}

	ctxWithTimeOut, ctxCancelFn := context.WithTimeout(ctx, 30*time.Second)
	defer ctxCancelFn()
	peerToSegment := map[string]*v2.HistorySegment{}
	for _, peerAddress := range peerAddresses {
		// We assume here that all/most datanodes will be running their GRPC API on the same port
		segment, err := GetMostRecentHistorySegmentFromPeer(ctxWithTimeOut, peerAddress, grpcAPIPort)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get most recent history segment:%w", err)
		}
		peerToSegment[peerAddress] = segment
	}

	if len(peerToSegment) == 0 {
		return nil, nil, ErrFailedToGetSegment
	}

	rootSegment := SelectRootSegment(peerToSegment)
	return rootSegment, peerToSegment, nil
}

func GetDatanodeBlockSpan(ctx context.Context, connConfig sqlstore.ConnectionConfig) (from int64, to int64, err error) {
	oldest, last, err := GetOldestHistoryBlockAndLastBlock(ctx, connConfig)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get oldest and last block:%w", err)
	}

	if oldest == nil || last == nil {
		return 0, 0, nil
	}

	return oldest.Height, last.Height, nil
}

func GetOldestHistoryBlockAndLastBlock(ctx context.Context, connConfig sqlstore.ConnectionConfig) (oldestHistoryBlock *entities.Block, lastBlock *entities.Block, err error) {
	hasVegaSchema, err := HasVegaSchema(ctx, connConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get check if database if empty:%w", err)
	}

	if hasVegaSchema {
		conn, err := pgxpool.Connect(ctx, connConfig.GetConnectionString())
		if err != nil {
			return nil, nil, fmt.Errorf("failed to connect to database: %w", err)
		}
		defer conn.Close()

		historyBlock, err := sqlstore.GetOldestHistoryBlockUsingConnection(ctx, conn)
		if err != nil {
			if !errors.Is(err, sqlstore.ErrNoHistoryBlock) {
				return nil, nil, fmt.Errorf("failed to get oldest history block:%w", err)
			}
		} else {
			oldestHistoryBlock = &historyBlock
		}

		block, err := sqlstore.GetLastBlockUsingConnection(ctx, conn)
		if err != nil {
			if !errors.Is(err, sqlstore.ErrNoLastBlock) {
				return nil, nil, fmt.Errorf("failed to get last block:%w", err)
			}
		} else {
			lastBlock = block
		}
	}

	return oldestHistoryBlock, lastBlock, nil
}

func DataNodeHasData(ctx context.Context, conf sqlstore.ConnectionConfig) (bool, error) {
	datanodeHasVegaSchema, err := HasVegaSchema(ctx, conf)
	if err != nil {
		return false, fmt.Errorf("failed to check if node has exisiting schema:%w", err)
	}

	datanodeIsEmpty := false
	if datanodeHasVegaSchema {
		datanodeIsEmpty, err = DataNodeIsEmpty(ctx, conf)
		if err != nil {
			return false, fmt.Errorf("failed to check if datanode is empty:%w", err)
		}
	}

	datanodeHasData := datanodeHasVegaSchema && !datanodeIsEmpty
	return datanodeHasData, nil
}

func HasVegaSchema(ctx context.Context, conf sqlstore.ConnectionConfig) (bool, error) {
	conn, err := pgxpool.Connect(ctx, conf.GetConnectionString())
	if err != nil {
		return false, fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close()

	tableNames, err := snapshot.GetAllTableNames(ctx, conn)
	if err != nil {
		return false, fmt.Errorf("failed to get all table names:%w", err)
	}

	return len(tableNames) != 0, nil
}

func DataNodeIsEmpty(ctx context.Context, connConfig sqlstore.ConnectionConfig) (bool, error) {
	datanodeFromHeight, datanodeToHeight, err := GetDatanodeBlockSpan(ctx, connConfig)
	if err != nil {
		return false, fmt.Errorf("failed to get datanode block span:%w", err)
	}
	return datanodeFromHeight == datanodeToHeight, nil
}

func GetMostRecentHistorySegmentFromPeer(ctx context.Context, ip string, datanodeGrpcAPIPort int) (*v2.HistorySegment, error) {
	client, conn, err := GetDatanodeClientFromIPAndPort(ip, datanodeGrpcAPIPort)
	if err != nil {
		return nil, fmt.Errorf("failed to get datanode client")
	}
	defer func() { _ = conn.Close() }()

	resp, err := client.GetMostRecentDeHistorySegment(ctx, &v2.GetMostRecentDeHistorySegmentRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get most recent history segment")
	}

	return resp.GetSegment(), nil
}

func GetDatanodeClientFromIPAndPort(ip string, port int) (v2.TradingDataServiceClient, *grpc.ClientConn, error) {
	address := net.JoinHostPort(ip, strconv.Itoa(port))
	tdconn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, nil, err
	}
	tradingDataClientV2 := v2.NewTradingDataServiceClient(&clientConn{tdconn})

	return tradingDataClientV2, tdconn, nil
}

type (
	clientConn struct {
		*grpc.ClientConn
	}
)

// TODO this needs some thought as to the best strategy to select the root segment to avoid spoofing.
func SelectRootSegment(peerToSegment map[string]*v2.HistorySegment) *v2.HistorySegment {
	segmentsList := make([]*v2.HistorySegment, 0, len(peerToSegment))

	for _, segment := range peerToSegment {
		segmentsList = append(segmentsList, segment)
	}

	// Sort history latest first
	sort.Slice(segmentsList, func(i, j int) bool {
		return segmentsList[i].ToHeight > segmentsList[j].ToHeight
	})

	// Filter out segments with toHeight < highest toHeight
	var filteredSegments []*v2.HistorySegment
	for _, segment := range segmentsList {
		if segment.ToHeight == segmentsList[0].ToHeight {
			filteredSegments = append(filteredSegments, segment)
		}
	}

	// Select one segment from the list at random
	rootSegment := filteredSegments[rand.Intn(len(filteredSegments))]
	return rootSegment
}

type FetchResult struct {
	HeightFrom               int64
	HeightTo                 int64
	PreviousHistorySegmentID string
}

func FromSegmentIndexEntry(s store.SegmentIndexEntry) FetchResult {
	return FetchResult{
		HeightFrom:               s.HeightFrom,
		HeightTo:                 s.HeightTo,
		PreviousHistorySegmentID: s.PreviousHistorySegmentID,
	}
}

// FetchHistoryBlocks will keep fetching history until numBlocksToFetch is reached or all history is retrieved.
func FetchHistoryBlocks(ctx context.Context, logInfo func(s string, args ...interface{}), historySegmentID string,
	fetchHistory func(ctx context.Context, historySegmentID string) (FetchResult, error),
	numBlocksToFetch int64,
) (int64, error) {
	blocksFetched := int64(0)
	for blocksFetched < numBlocksToFetch {
		logInfo("fetching history for segment id:%s", historySegmentID)
		indexEntry, err := fetchHistory(ctx, historySegmentID)
		if err != nil {
			return 0, fmt.Errorf("failed to fetch history:%w", err)
		}
		blocksFetched += indexEntry.HeightTo - indexEntry.HeightFrom + 1

		logInfo("fetched history:%+v", indexEntry)

		if len(indexEntry.PreviousHistorySegmentID) == 0 {
			break
		}

		historySegmentID = indexEntry.PreviousHistorySegmentID
	}

	return blocksFetched, nil
}

func VerifyChainID(chainID string, chainService *service.Chain) error {
	if len(chainID) == 0 {
		return errors.New("chain id must be set")
	}

	currentChainID, err := chainService.GetChainID()
	if err != nil {
		if errors.Is(err, entities.ErrChainNotFound) {
			return ErrChainNotFound
		}

		return fmt.Errorf("failed to get chain id:%w", err)
	}

	if len(currentChainID) == 0 {
		chainService.SetChainID(chainID)
	} else if currentChainID != chainID {
		return fmt.Errorf("mismatched chain ids, config chain id: %s, current chain id: %s", chainID, currentChainID)
	}
	return nil
}
