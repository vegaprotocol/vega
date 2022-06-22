// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package api_test

import (
	"context"
	"testing"
	"time"

	apipb "code.vegaprotocol.io/protos/data-node/api/v1"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOracleSpecs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	server := NewTestServer(t, ctx, true)
	defer server.ctrl.Finish()

	PublishEvents(t, ctx, server.broker, func(be *eventspb.BusEvent) (events.Event, error) {
		spec := be.GetOracleSpec()
		require.NotNil(t, spec)
		return events.NewOracleSpecEvent(ctx, oraclespb.OracleSpec{
			Id:        spec.Id,
			CreatedAt: spec.CreatedAt,
			UpdatedAt: spec.UpdatedAt,
			PubKeys:   spec.PubKeys,
			Filters:   spec.Filters,
			Status:    spec.Status,
		}), nil
	}, "oracle-spec-events.golden")

	client := apipb.NewTradingDataServiceClient(server.clientConn)
	require.NotNil(t, client)

	oracleSpecID := "6f9b102855efc7b2421df3de4007bd3c6b9fd237e0f9b9b18326800fd822184f"

	var resp *apipb.OracleSpecsResponse
	var err error

	ticker := time.NewTicker(50 * time.Millisecond)

loop:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("test timeout")
		case <-ticker.C:
			resp, err = client.OracleSpecs(ctx, &apipb.OracleSpecsRequest{})
			require.NotNil(t, resp)
			require.NoError(t, err)
			if len(resp.OracleSpecs) > 0 {
				break loop
			}
		}
	}

	assert.NoError(t, err)
	assert.Equal(t, oracleSpecID, resp.OracleSpecs[0].Id)
}

func TestOracleDataBySpec(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	server := NewTestServer(t, ctx, true)
	defer server.ctrl.Finish()

	PublishEvents(t, ctx, server.broker, func(be *eventspb.BusEvent) (events.Event, error) {
		data := be.GetOracleData()
		require.NotNil(t, data)
		return events.NewOracleDataEvent(ctx, oraclespb.OracleData{
			PubKeys:        data.PubKeys,
			Data:           data.Data,
			MatchedSpecIds: data.MatchedSpecIds,
			BroadcastAt:    data.BroadcastAt,
		}), nil
	}, "oracle-data-events.golden")

	client := apipb.NewTradingDataServiceClient(server.clientConn)
	require.NotNil(t, client)

	var resp *apipb.OracleDataBySpecResponse
	var err error

	ticker := time.NewTicker(50 * time.Millisecond)

loop:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("test timeout")
		case <-ticker.C:
			resp, err = client.OracleDataBySpec(ctx, &apipb.OracleDataBySpecRequest{
				Id: "1234567890",
			})
			require.NotNil(t, resp)
			require.NoError(t, err)
			if len(resp.OracleData) > 0 {
				break loop
			}
		}
	}

	require.NoError(t, err)
	require.Len(t, resp.OracleData, 1)
	assert.Equal(t, &oraclespb.OracleData{
		PubKeys: []string{"0xdeadbeef"},
		Data: []*oraclespb.Property{
			{
				Name:  "hello",
				Value: "world",
			},
		},
		MatchedSpecIds: []string{"1234567890", "0987654321"},
		BroadcastAt:    1652696804,
	}, resp.OracleData[0])
}

func TestOracleDataBySpecWhenNotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	server := NewTestServer(t, ctx, true)
	defer server.ctrl.Finish()

	PublishEvents(t, ctx, server.broker, func(be *eventspb.BusEvent) (events.Event, error) {
		data := be.GetOracleData()
		require.NotNil(t, data)
		return events.NewOracleDataEvent(ctx, oraclespb.OracleData{
			PubKeys:        data.PubKeys,
			Data:           data.Data,
			MatchedSpecIds: data.MatchedSpecIds,
			BroadcastAt:    data.BroadcastAt,
		}), nil
	}, "oracle-data-events.golden")

	client := apipb.NewTradingDataServiceClient(server.clientConn)
	require.NotNil(t, client)

	var resp *apipb.OracleDataBySpecResponse
	var err error

	ticker := time.NewTicker(50 * time.Millisecond)

loop:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("test timeout")
		case <-ticker.C:
			resp, err = client.OracleDataBySpec(ctx, &apipb.OracleDataBySpecRequest{
				Id: "qwertyu",
			})
			require.Nil(t, resp)
			if err != nil {
				break loop
			}
		}
	}

	require.Error(t, err)
}

func TestListOracleData(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	server := NewTestServer(t, ctx, true)
	defer server.ctrl.Finish()

	PublishEvents(t, ctx, server.broker, func(be *eventspb.BusEvent) (events.Event, error) {
		data := be.GetOracleData()
		require.NotNil(t, data)
		return events.NewOracleDataEvent(ctx, oraclespb.OracleData{
			PubKeys:        data.PubKeys,
			Data:           data.Data,
			MatchedSpecIds: data.MatchedSpecIds,
			BroadcastAt:    data.BroadcastAt,
		}), nil
	}, "oracle-data-events.golden")

	client := apipb.NewTradingDataServiceClient(server.clientConn)
	require.NotNil(t, client)

	var resp *apipb.ListOracleDataResponse
	var err error

	ticker := time.NewTicker(50 * time.Millisecond)

loop:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("test timeout")
		case <-ticker.C:
			resp, err = client.ListOracleData(ctx, &apipb.ListOracleDataRequest{})
			require.NotNil(t, resp)
			require.NoError(t, err)
			if len(resp.OracleData) > 0 {
				break loop
			}
		}
	}

	require.NoError(t, err)
	require.Len(t, resp.OracleData, 3)
	assert.Equal(t, &oraclespb.OracleData{
		PubKeys: []string{"0x00000000"},
		Data: []*oraclespb.Property{
			{
				Name:  "jane",
				Value: "doe",
			},
		},
		MatchedSpecIds: []string{"0987654321"},
		BroadcastAt:    1652696805,
	}, resp.OracleData[0])
	assert.Equal(t, &oraclespb.OracleData{
		PubKeys: []string{"0xdeadbeef"},
		Data: []*oraclespb.Property{
			{
				Name:  "hello",
				Value: "world",
			},
		},
		MatchedSpecIds: []string{"1234567890", "0987654321"},
		BroadcastAt:    1652696804,
	}, resp.OracleData[1])
	assert.Equal(t, &oraclespb.OracleData{
		PubKeys: []string{"0xcafed00d"},
		Data: []*oraclespb.Property{
			{
				Name:  "john",
				Value: "doe",
			},
		},
		MatchedSpecIds: nil,
		BroadcastAt:    0,
	}, resp.OracleData[2])
}
