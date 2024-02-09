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

package events

import (
	"context"
	"fmt"
	"os"
	"time"

	"code.vegaprotocol.io/vega/datanode/broker"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

func Run(in, out string) error {
	marshaler := jsonpb.Marshaler{
		EnumsAsInts: true,
		OrigName:    true,
		Indent:      "   ",
	}

	fmt.Println("parsing event bytes from", in, "into json:", out)
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()

	eventFile, err := os.Open(in)
	if err != nil {
		return err
	}
	defer eventFile.Close()

	ctx, cfunc := context.WithCancel(context.Background())
	defer cfunc()

	ch, ech := startFileRead(ctx, eventFile)

	for {
		select {
		case e, ok := <-ch:
			if e == nil && !ok {
				return nil
			}
			es, err := marshaler.MarshalToString(e)
			if err != nil {
				return err
			}
			if _, err := f.WriteString(es + "\n"); err != nil {
				return err
			}
		case err, ok := <-ech:
			if err == nil && !ok {
				return nil
			}
			return err
		}
	}
}

func startFileRead(ctx context.Context, eventFile *os.File) (<-chan *eventspb.BusEvent, <-chan error) {
	ch := make(chan *eventspb.BusEvent, 1)
	ech := make(chan error, 1)
	go func() {
		defer func() {
			eventFile.Close()
			close(ch)
			close(ech)
		}()

		var offset, nEvents int64
		now := time.Now()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				rawEvent, _, read, err := broker.ReadRawEvent(eventFile, offset)
				if err != nil {
					ech <- fmt.Errorf("failed to read raw event: %w", err)
					return
				}

				if read == 0 {
					return
				}

				offset += int64(read)
				busEvent := &eventspb.BusEvent{}
				if err := proto.Unmarshal(rawEvent, busEvent); err != nil {
					ech <- fmt.Errorf("failed to unmarshal bus event: %w", err)
					return
				}
				ch <- busEvent
				// if can be quite slow so lets print something out every now and again so it doesn't look like its frozen
				nEvents++
				if time.Since(now) > time.Second {
					fmt.Println("events parsed so far:", nEvents)
					now = time.Now()
				}
			}
		}
	}()
	return ch, ech
}
