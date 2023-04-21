package api_test

import (
	"archive/zip"
	"bytes"
	"context"
	"embed"
	"io"
	"strings"
	"testing"

	"code.vegaprotocol.io/vega/datanode/api"
	"code.vegaprotocol.io/vega/datanode/api/mocks"
	"code.vegaprotocol.io/vega/datanode/networkhistory/segment"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/api/httpbody"
	"google.golang.org/grpc/metadata"
)

//go:embed testdata/dummysegment.zip
var testData embed.FS

func makeFullSegment(from, to, dbVersion int64) segment.Full {
	return segment.Full{
		MetaData: segment.MetaData{
			Base: segment.Base{
				HeightFrom:      from,
				HeightTo:        to,
				DatabaseVersion: dbVersion,
				ChainID:         "test-chain-id",
			},
		},
	}
}

func TestExportNetworkHistory(t *testing.T) {
	req := &v2.ExportNetworkHistoryRequest{
		FromBlock: 1,
		ToBlock:   3000,
		Table:     v2.Table_TABLE_ORDERS,
	}

	ctrl := gomock.NewController(t)
	historyService := mocks.NewMockNetworkHistoryService(ctrl)

	testSegments := []segment.Full{
		makeFullSegment(1, 1000, 1),
		makeFullSegment(1001, 2000, 1),
		makeFullSegment(2001, 3000, 2),
	}

	historyService.EXPECT().ListAllHistorySegments().Times(1).Return(testSegments, nil)
	historyService.EXPECT().GetHistorySegmentReader(gomock.Any(), gomock.Any()).Times(3).DoAndReturn(
		func(ctx context.Context, id string) (io.ReadSeekCloser, int64, error) {
			reader, err := testData.Open("testdata/dummysegment.zip")
			require.NoError(t, err)
			info, _ := reader.Stat()
			return reader.(io.ReadSeekCloser), info.Size(), nil
		},
	)

	stream := &mockStream{}
	apiService := api.TradingDataServiceV2{
		NetworkHistoryService: historyService,
	}

	err := apiService.ExportNetworkHistory(req, stream)
	require.NoError(t, err)

	// Now check that we got a zip file with two CSV files in it; as we crossed a schema migration boundary
	require.Greater(t, len(stream.sent), 0)
	assert.Equal(t, stream.sent[0].ContentType, "application/zip")

	zipBytes := stream.sent[0].Data
	zipBuffer := bytes.NewReader(zipBytes)
	zipReader, err := zip.NewReader(zipBuffer, int64(len(zipBytes)))
	require.NoError(t, err)

	filenames := []string{}
	for _, file := range zipReader.File {
		filenames = append(filenames, file.Name)
		fileReader, err := file.Open()
		require.NoError(t, err)
		fileContents, err := io.ReadAll(fileReader)
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(string(fileContents), "header row\nmock data, more mock data,"))
	}

	require.Equal(t, filenames, []string{
		"test-chain-id-orders-001-000001-002000.csv",
		"test-chain-id-orders-002-002001-003000.csv",
	})
}

type mockStream struct {
	sent []*httpbody.HttpBody
}

func (s *mockStream) Send(b *httpbody.HttpBody) error { s.sent = append(s.sent, b); return nil }
func (s *mockStream) SetHeader(metadata.MD) error     { return nil }
func (s *mockStream) SendHeader(metadata.MD) error    { return nil }
func (s *mockStream) SetTrailer(metadata.MD)          {}
func (s *mockStream) Context() context.Context        { return context.Background() }
func (s *mockStream) SendMsg(m interface{}) error     { return nil }
func (s *mockStream) RecvMsg(m interface{}) error     { return nil }
