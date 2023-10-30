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

package admin

import (
	"context"
	"fmt"
	"net/http"

	"code.vegaprotocol.io/vega/datanode/networkhistory/segment"
)

type NetworkHistoryService interface {
	CopyHistorySegmentToFile(ctx context.Context, historySegmentID string, outFile string) error
	FetchHistorySegment(ctx context.Context, historySegmentID string) (segment.Full, error)
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

func (d *NetworkHistoryAdminService) FetchHistorySegment(req *http.Request, historySegmentID *string, reply *segment.Full) (err error) {
	*reply, err = d.networkHistoryService.FetchHistorySegment(req.Context(), *historySegmentID)
	return
}
