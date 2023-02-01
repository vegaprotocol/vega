package admin

import (
	"context"
	"fmt"
	"net/http"

	"code.vegaprotocol.io/vega/datanode/networkhistory"
)

type NetworkHistoryService interface {
	CopyHistorySegmentToFile(ctx context.Context, historySegmentID string, outFile string) error
	FetchHistorySegment(ctx context.Context, historySegmentID string) (networkhistory.Segment, error)
}

type NetworkHistoryAdminService struct {
	networkHistoryService NetworkHistoryService
}

type CopyHistorySegmentToFileArg struct {
	HistorySegmentID string
	OutFile          string
}

type CopyHistorySegmentToFileReply struct {
	Reply string
	Err   error
}

func NewNetworkHistoryAdminService(networkHistoryService NetworkHistoryService) *NetworkHistoryAdminService {
	return &NetworkHistoryAdminService{
		networkHistoryService: networkHistoryService,
	}
}

func (d *NetworkHistoryAdminService) CopyHistorySegmentToFile(req *http.Request, args *CopyHistorySegmentToFileArg, reply *CopyHistorySegmentToFileReply) error {
	err := d.networkHistoryService.CopyHistorySegmentToFile(req.Context(), args.HistorySegmentID, args.OutFile)
	if err != nil {
		reply.Err = fmt.Errorf("copy history segment %s to file %s failed - %w", args.HistorySegmentID, args.OutFile, err)
		return err
	}

	reply.Reply = fmt.Sprintf("copied history segment %s to file %s", args.HistorySegmentID, args.OutFile)
	return err
}

func (d *NetworkHistoryAdminService) FetchHistorySegment(req *http.Request, historySegmentID *string, reply *networkhistory.Segment) (err error) {
	*reply, err = d.networkHistoryService.FetchHistorySegment(req.Context(), *historySegmentID)
	return
}
