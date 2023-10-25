// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
	"code.vegaprotocol.io/vega/datanode/networkhistory/segment"
	"code.vegaprotocol.io/vega/datanode/networkhistory/snapshot"
	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"google.golang.org/grpc"
)

var ErrChainNotFound = errors.New("no chain found")

// it would be nice to use go:generate go run github.com/golang/mock/mockgen -destination mocks/networkhistory_service_mock.go -package mocks code.vegaprotocol.io/vega/datanode/networkhistory NetworkHistory
// but it messes up with generic interfaces and so requires a bit of manual fiddling.
type NetworkHistory interface {
	FetchHistorySegment(ctx context.Context, historySegmentID string) (segment.Full, error)
	LoadNetworkHistoryIntoDatanode(ctx context.Context, chunk segment.ContiguousHistory[segment.Full], cfg sqlstore.ConnectionConfig, withIndexesAndOrderTriggers, verbose bool) (snapshot.LoadResult, error)
	GetMostRecentHistorySegmentFromBootstrapPeers(ctx context.Context, grpcAPIPorts []int) (*PeerResponse, map[string]*v2.GetMostRecentNetworkHistorySegmentResponse, error)
	GetDatanodeBlockSpan(ctx context.Context) (sqlstore.DatanodeBlockSpan, error)
	ListAllHistorySegments() (segment.Segments[segment.Full], error)
}

func InitialiseDatanodeFromNetworkHistory(ctx context.Context, cfg InitializationConfig, log *logging.Logger,
	connCfg sqlstore.ConnectionConfig, networkHistoryService NetworkHistory,
	grpcPorts []int, verboseMigration bool,
) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, cfg.TimeOut.Duration)
	defer cancel()

	if len(cfg.ToSegment) == 0 {
		for {
			mostRecentHistorySegment, err := getMostRecentNetworkHistorySegment(ctxWithTimeout, networkHistoryService, grpcPorts, log)
			if err != nil {
				return fmt.Errorf("failed to get most recent history segment: %w", err)
			}

			toSegmentID := mostRecentHistorySegment.HistorySegmentId

			currentSpan, err := networkHistoryService.GetDatanodeBlockSpan(ctxWithTimeout)
			if err != nil {
				return fmt.Errorf("failed to get datanode block span: %w", err)
			}

			var blocksToFetch int64
			if currentSpan.HasData {
				if currentSpan.ToHeight >= mostRecentHistorySegment.ToHeight {
					log.Infof("data node height %d is already at or beyond the height of the most recent history segment %d, no further history to load",
						currentSpan.ToHeight, mostRecentHistorySegment.ToHeight)
					return nil
				}

				blocksToFetch = mostRecentHistorySegment.ToHeight - currentSpan.ToHeight
			} else {
				// check if goes < 0
				blocksToFetch = cfg.MinimumBlockCount
				if mostRecentHistorySegment.ToHeight-cfg.MinimumBlockCount < 0 {
					blocksToFetch = -1
				}
			}

			err = loadSegments(ctxWithTimeout, log, connCfg, networkHistoryService, currentSpan,
				toSegmentID, blocksToFetch, verboseMigration)
			if err != nil {
				return fmt.Errorf("failed to load segments: %w", err)
			}
		}
	} else {
		currentSpan, err := networkHistoryService.GetDatanodeBlockSpan(ctx)
		if err != nil {
			return fmt.Errorf("failed to get datanode block span: %w", err)
		}

		err = loadSegments(ctxWithTimeout, log, connCfg, networkHistoryService, currentSpan,
			cfg.ToSegment, cfg.MinimumBlockCount, verboseMigration)
		if err != nil {
			return fmt.Errorf("failed to load segments: %w", err)
		}
	}

	return nil
}

func loadSegments(ctx context.Context, log *logging.Logger,
	connCfg sqlstore.ConnectionConfig, networkHistoryService NetworkHistory, currentSpan sqlstore.DatanodeBlockSpan, toSegmentID string, blocksToFetch int64,
	verboseMigration bool,
) error {
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
		log.Errorf("failed to fetch history blocks: %v", err)
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

	chunks := segments.AllContigousHistories()
	if len(chunks) == 0 {
		log.Infof("no network history available to load")
		return nil
	}

	lastChunk, err := segments.MostRecentContiguousHistory()
	if err != nil {
		return fmt.Errorf("failed to get most recent chunk")
	}

	if currentSpan.ToHeight >= lastChunk.HeightTo {
		log.Infof("datanode already contains the latest network history data")
		return nil
	}

	to := lastChunk.HeightTo
	from := lastChunk.HeightFrom
	if currentSpan.HasData {
		for _, segment := range lastChunk.Segments {
			if segment.GetFromHeight() <= (currentSpan.ToHeight+1) && segment.GetToHeight() > currentSpan.ToHeight {
				from = segment.GetFromHeight()
				break
			}
		}
	}

	chunkToLoad, err := segments.ContiguousHistoryInRange(from, to)
	if err != nil {
		return fmt.Errorf("failed to load history into the datanode: %w", err)
	}

	loaded, err := networkHistoryService.LoadNetworkHistoryIntoDatanode(ctx, chunkToLoad, connCfg, currentSpan.HasData, verboseMigration)
	if err != nil {
		return fmt.Errorf("failed to load history into the datanode: %w", err)
	}
	log.Infof("loaded history from height %d to %d into the datanode", loaded.LoadedFromHeight, loaded.LoadedToHeight)

	return nil
}

func getMostRecentNetworkHistorySegment(ctx context.Context, networkHistoryService NetworkHistory, grpcPorts []int, log *logging.Logger) (*v2.HistorySegment, error) {
	response, _, err := networkHistoryService.GetMostRecentHistorySegmentFromBootstrapPeers(ctx,
		grpcPorts)
	if err != nil {
		log.Errorf("failed to get most recent history segment from peers: %v", err)
		return nil, fmt.Errorf("failed to get most recent history segment from peers: %w", err)
	}

	if response == nil {
		log.Error("unable to get a most recent segment response from peers")
		return nil, errors.New("unable to get a most recent segment response from peers")
	}

	mostRecentHistorySegment := response.Response.Segment

	log.Info("got most recent history segment",
		logging.String("segment", mostRecentHistorySegment.String()), logging.String("peer", response.PeerAddr))
	return mostRecentHistorySegment, nil
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

func FromSegmentIndexEntry(s segment.Full) FetchResult {
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
	for blocksFetched < numBlocksToFetch || numBlocksToFetch == -1 {
		logInfo("fetching history for segment id %q", historySegmentID)
		indexEntry, err := fetchHistory(ctx, historySegmentID)
		if err != nil {
			return 0, fmt.Errorf("failed to fetch history:%w", err)
		}
		blocksFetched += indexEntry.HeightTo - indexEntry.HeightFrom + 1

		logInfo("fetched history: %+v", indexEntry)

		if len(indexEntry.PreviousHistorySegmentID) == 0 {
			break
		}

		historySegmentID = indexEntry.PreviousHistorySegmentID
		if len(historySegmentID) == 0 {
			break
		}
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
