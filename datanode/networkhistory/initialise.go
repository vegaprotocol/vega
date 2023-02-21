package networkhistory

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/networkhistory/snapshot"
	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"google.golang.org/grpc"
)

var ErrChainNotFound = errors.New("no chain found")

//go:generate go run github.com/golang/mock/mockgen -destination mocks/networkhistory_service_mock.go -package mocks code.vegaprotocol.io/vega/datanode/networkhistory NetworkHistory
type NetworkHistory interface {
	FetchHistorySegment(ctx context.Context, historySegmentID string) (Segment, error)
	LoadNetworkHistoryIntoDatanode(ctx context.Context, contiguousHistory ContiguousHistory, cfg sqlstore.ConnectionConfig, withIndexesAndOrderTriggers, verbose bool) (snapshot.LoadResult, error)
	GetMostRecentHistorySegmentFromPeers(ctx context.Context, grpcAPIPorts []int) (*PeerResponse, map[string]*v2.GetMostRecentNetworkHistorySegmentResponse, error)
	GetDatanodeBlockSpan(ctx context.Context) (sqlstore.DatanodeBlockSpan, error)
	ListAllHistorySegments() ([]Segment, error)
}

func InitialiseDatanodeFromNetworkHistory(parentCtx context.Context, cfg InitializationConfig, log *logging.Logger,
	connCfg sqlstore.ConnectionConfig, networkHistoryService NetworkHistory,
	grpcPorts []int, verboseMigration bool,
) error {
	ctx, ctxCancelFn := context.WithTimeout(parentCtx, cfg.TimeOut.Duration)
	defer ctxCancelFn()

	currentSpan, err := networkHistoryService.GetDatanodeBlockSpan(ctx)
	if err != nil {
		return fmt.Errorf("failed to get datanode block span: %w", err)
	}

	var toSegmentID string
	blocksToFetch := cfg.MinimumBlockCount
	if len(cfg.ToSegment) == 0 {
		response, _, err := networkHistoryService.GetMostRecentHistorySegmentFromPeers(ctx,
			grpcPorts)
		if err != nil {
			return fmt.Errorf("failed to get most recent history segment from peers: %w", err)
		}

		if response == nil {
			return errors.New("unable to get a most recent segment response from peers")
		}

		mostRecentHistorySegment := response.Response.Segment

		log.Info("got most recent history segment",
			logging.String("segment", mostRecentHistorySegment.String()), logging.String("peer", response.PeerAddr))

		toSegmentID = mostRecentHistorySegment.HistorySegmentId

		if currentSpan.HasData {
			if currentSpan.ToHeight >= mostRecentHistorySegment.ToHeight {
				log.Infof("data node height %d is already at or beyond the height of the most recent history segment %d, not loading any history",
					currentSpan.ToHeight, mostRecentHistorySegment.ToHeight)
				return nil
			}

			blocksToFetch = mostRecentHistorySegment.ToHeight - currentSpan.ToHeight
		}
	} else {
		toSegmentID = cfg.ToSegment
	}

	log.Infof("fetching history using as the first segment:{%s} and minimum blocks to fetch %d", toSegmentID, blocksToFetch)

	blocksFetched, err := FetchHistoryBlocks(ctx, log.Infof, toSegmentID,
		func(ctx context.Context, historySegmentID string) (FetchResult, error) {
			segment, err := networkHistoryService.FetchHistorySegment(ctx, historySegmentID)
			if err != nil {
				return FetchResult{}, err
			}
			return FromSegmentIndexEntry(segment), nil
		}, blocksToFetch)
	if err != nil {
		return fmt.Errorf("failed to fetch history blocks:%w", err)
	}

	if blocksFetched == 0 {
		return fmt.Errorf("failed to get any blocks from network history")
	}

	log.Infof("fetched %d blocks from network history", blocksFetched)

	log.Infof("loading history into the datanode")
	segments, err := networkHistoryService.ListAllHistorySegments()
	if err != nil {
		return fmt.Errorf("failed to list all history segments: %w", err)
	}

	contiguousHistories := GetContiguousHistories(segments)
	if len(contiguousHistories) == 0 {
		log.Infof("no network history available to load")
		return nil
	}

	latestContiguousHistory := contiguousHistories[len(contiguousHistories)-1]
	if currentSpan.ToHeight >= latestContiguousHistory.HeightTo {
		log.Infof("datanode already contains the latest network history data")
		return nil
	}

	to := latestContiguousHistory.HeightTo
	from := latestContiguousHistory.HeightFrom
	if currentSpan.HasData {
		for _, segment := range latestContiguousHistory.SegmentsOldestFirst {
			if segment.GetFromHeight() <= currentSpan.ToHeight && segment.GetToHeight() > currentSpan.ToHeight {
				from = segment.GetFromHeight()
				break
			}
		}
	}

	contiguousHistory := GetContiguousHistoryForSpan(contiguousHistories, from, to)

	loaded, err := networkHistoryService.LoadNetworkHistoryIntoDatanode(ctx, *contiguousHistory, connCfg, currentSpan.HasData, verboseMigration)
	if err != nil {
		return fmt.Errorf("failed to load history into the datanode: %w", err)
	}
	log.Infof("loaded history from height %d to %d into the datanode", loaded.LoadedFromHeight, loaded.LoadedToHeight)

	return nil
}

func VerifyChainID(chainID string, chainService *service.Chain) error {
	if len(chainID) == 0 {
		return errors.New("chain id must be set")
	}

	currentChainID, err := chainService.GetChainID()
	if err != nil {
		if errors.Is(err, entities.ErrNotFound) {
			return ErrChainNotFound
		}

		return fmt.Errorf("failed to get chain id:%w", err)
	}

	if len(currentChainID) == 0 {
		if err = chainService.SetChainID(chainID); err != nil {
			return fmt.Errorf("failed to set chain id:%w", err)
		}
	} else if currentChainID != chainID {
		return fmt.Errorf("mismatched chain ids, config chain id: %s, current chain id: %s", chainID, currentChainID)
	}
	return nil
}

type FetchResult struct {
	HeightFrom               int64
	HeightTo                 int64
	PreviousHistorySegmentID string
}

func FromSegmentIndexEntry(s Segment) FetchResult {
	return FetchResult{
		HeightFrom:               s.GetFromHeight(),
		HeightTo:                 s.GetToHeight(),
		PreviousHistorySegmentID: s.GetPreviousHistorySegmentId(),
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

type PeerResponse struct {
	PeerAddr string
	Response *v2.GetMostRecentNetworkHistorySegmentResponse
}

func GetMostRecentHistorySegmentFromPeersAddresses(ctx context.Context, peerAddresses []string,
	swarmKeySeed string,
	grpcAPIPorts []int,
) (*PeerResponse, map[string]*v2.GetMostRecentNetworkHistorySegmentResponse, error) {
	const maxPeersToContact = 10

	if len(peerAddresses) > maxPeersToContact {
		peerAddresses = peerAddresses[:maxPeersToContact]
	}

	ctxWithTimeOut, ctxCancelFn := context.WithTimeout(ctx, 30*time.Second)
	defer ctxCancelFn()
	peerToResponse := map[string]*v2.GetMostRecentNetworkHistorySegmentResponse{}
	var errorMsgs []string
	for _, peerAddress := range peerAddresses {
		for _, grpcAPIPort := range grpcAPIPorts {
			resp, err := GetMostRecentHistorySegmentFromPeer(ctxWithTimeOut, peerAddress, grpcAPIPort)
			if err == nil {
				peerAddress = net.JoinHostPort(peerAddress, strconv.Itoa(grpcAPIPort))
				peerToResponse[peerAddress] = resp
			} else {
				errorMsgs = append(errorMsgs, err.Error())
			}
		}
	}

	if len(peerToResponse) == 0 {
		return nil, nil, fmt.Errorf(strings.Join(errorMsgs, ","))
	}

	return SelectMostRecentHistorySegmentResponse(peerToResponse, swarmKeySeed), peerToResponse, nil
}

func GetMostRecentHistorySegmentFromPeer(ctx context.Context, ip string, datanodeGrpcAPIPort int) (*v2.GetMostRecentNetworkHistorySegmentResponse, error) {
	client, conn, err := GetDatanodeClientFromIPAndPort(ip, datanodeGrpcAPIPort)
	if err != nil {
		return nil, fmt.Errorf("failed to get datanode client:%w", err)
	}
	defer func() { _ = conn.Close() }()

	resp, err := client.GetMostRecentNetworkHistorySegment(ctx, &v2.GetMostRecentNetworkHistorySegmentRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get most recent history segment:%w", err)
	}

	return resp, nil
}

// TODO this needs some thought as to the best strategy to select the response to avoid spoofing.
func SelectMostRecentHistorySegmentResponse(peerToResponse map[string]*v2.GetMostRecentNetworkHistorySegmentResponse, swarmKeySeed string) *PeerResponse {
	responses := make([]PeerResponse, 0, len(peerToResponse))

	highestResponseHeight := int64(0)
	for peer, response := range peerToResponse {
		if response.SwarmKeySeed == swarmKeySeed {
			responses = append(responses, PeerResponse{peer, response})

			if response.Segment.ToHeight > highestResponseHeight {
				highestResponseHeight = response.Segment.ToHeight
			}
		}
	}

	var responsesAtHighestHeight []PeerResponse
	for _, response := range responses {
		if response.Response.Segment.ToHeight == highestResponseHeight {
			responsesAtHighestHeight = append(responsesAtHighestHeight, response)
		}
	}

	// Select one response from the list at random
	if len(responsesAtHighestHeight) > 0 {
		segment := responsesAtHighestHeight[rand.Intn(len(responsesAtHighestHeight))]
		return &segment
	}

	return nil
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
