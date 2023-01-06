package admin

import (
	"context"
	"fmt"
	"net/http"

	"code.vegaprotocol.io/vega/datanode/dehistory/store"
)

type DeHistoryService interface {
	CopyHistorySegmentToFile(ctx context.Context, historySegmentID string, outFile string) error
	FetchHistorySegment(ctx context.Context, historySegmentID string) (store.SegmentIndexEntry, error)
}

type DeHistoryAdminService struct {
	deHistoryService DeHistoryService
}

type CopyHistorySegmentToFileArg struct {
	HistorySegmentID string
	OutFile          string
}

type CopyHistorySegmentToFileReply struct {
	Reply string
	Err   error
}

func NewDeHistoryAdminService(deHistoryService DeHistoryService) *DeHistoryAdminService {
	return &DeHistoryAdminService{
		deHistoryService: deHistoryService,
	}
}

func (d *DeHistoryAdminService) CopyHistorySegmentToFile(req *http.Request, args *CopyHistorySegmentToFileArg, reply *CopyHistorySegmentToFileReply) error {
	err := d.deHistoryService.CopyHistorySegmentToFile(req.Context(), args.HistorySegmentID, args.OutFile)
	if err != nil {
		reply.Err = fmt.Errorf("copy history segment %s to file %s failed - %w", args.HistorySegmentID, args.OutFile, err)
		return err
	}

	reply.Reply = fmt.Sprintf("copied history segment %s to file %s", args.HistorySegmentID, args.OutFile)
	return err
}

func (d *DeHistoryAdminService) FetchHistorySegment(req *http.Request, historySegmentID *string, reply *store.SegmentIndexEntry) (err error) {
	*reply, err = d.deHistoryService.FetchHistorySegment(req.Context(), *historySegmentID)
	return
}
