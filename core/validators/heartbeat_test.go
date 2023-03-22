// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package validators

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/stretchr/testify/require"
)

func TestRecordHeartbeatResult(t *testing.T) {
	top := getHBTestTopology(t)
	tracker := top.validators["node1"].heartbeatTracker

	for i := 0; i < 100; i++ {
		if i%2 == 0 {
			tracker.recordHeartbeatResult(true)
			require.Equal(t, true, tracker.blockSigs[i%10])
		} else {
			tracker.recordHeartbeatResult(false)
			require.Equal(t, false, tracker.blockSigs[i%10])
		}
		require.Equal(t, "", tracker.expectedNextHash)
		require.Equal(t, time.Time{}, tracker.expectedNexthashSince)
	}
}

func TestCheckAndExpireStaleHeartbeats(t *testing.T) {
	top := getHBTestTopology(t)
	top.epochSeq = 0
	tt := &dummyTestTime{}
	top.timeService = tt

	now := time.Now()
	nowPlus500 := now.Add(500 * time.Second)
	tt.now = now

	// no next hash - means we're not awaiting a heartbeat, nothing expired
	top.checkAndExpireStaleHeartbeats()
	require.Equal(t, 0, top.validators["node1"].heartbeatTracker.blockIndex)

	top.validators["node1"].heartbeatTracker.expectedNextHash = "abcde"
	top.validators["node1"].heartbeatTracker.expectedNexthashSince = now
	top.checkAndExpireStaleHeartbeats()
	require.Equal(t, 0, top.validators["node1"].heartbeatTracker.blockIndex)

	// still not enough time passed
	tt.now = nowPlus500
	top.validators["node1"].heartbeatTracker.expectedNextHash = "abcde"
	top.validators["node1"].heartbeatTracker.expectedNexthashSince = now
	top.checkAndExpireStaleHeartbeats()
	require.Equal(t, 0, top.validators["node1"].heartbeatTracker.blockIndex)

	// enough time passed - expect invalidation
	tt.now = nowPlus500.Add(1 * time.Second)
	top.validators["node1"].heartbeatTracker.expectedNextHash = "abcde"
	top.validators["node1"].heartbeatTracker.expectedNexthashSince = now
	top.checkAndExpireStaleHeartbeats()
	require.Equal(t, 1, top.validators["node1"].heartbeatTracker.blockIndex)
	require.Equal(t, "", top.validators["node1"].heartbeatTracker.expectedNextHash)

	// set up a new expected heartbeat and pretend the old one arrived again
	cmd := &commandspb.ValidatorHeartbeat{
		NodeId:  "node1",
		Message: "abcde",
	}
	top.validators["node1"].heartbeatTracker.expectedNextHash = "abcdeagain"
	top.validators["node1"].heartbeatTracker.expectedNexthashSince = now
	require.Error(t, ErrHeartbeatHasExpired, top.ProcessValidatorHeartbeat(context.Background(), cmd, nil, nil))
	require.Equal(t, "abcdeagain", top.validators["node1"].heartbeatTracker.expectedNextHash)
}

func TestProcessValidatorHeartbeat(t *testing.T) {
	topology := &Topology{}
	topology.validators = map[string]*valState{}
	cmd := &commandspb.ValidatorHeartbeat{}
	cmd.NodeId = "node1"

	// invalid node ID
	require.Error(t, topology.ProcessValidatorHeartbeat(context.Background(), cmd, nil, nil))

	topology.validators["node1"] = &valState{
		heartbeatTracker: &validatorHeartbeatTracker{},
	}

	for i := 0; i < 10; i++ {
		topology.validators["node1"].heartbeatTracker.blockSigs[i] = true
	}

	// undecodable signature
	cmd.VegaSignature = &commandspb.Signature{
		Value: "haha",
	}

	require.Error(t, topology.ProcessValidatorHeartbeat(context.Background(), cmd, nil, nil))
	require.Equal(t, 1, topology.validators["node1"].heartbeatTracker.blockIndex)
	require.Equal(t, false, topology.validators["node1"].heartbeatTracker.blockSigs[0])

	cmd.VegaSignature.Value = "abcdef"

	topology.validators["node1"].data.VegaPubKey = "fooo"
	require.Error(t, topology.ProcessValidatorHeartbeat(context.Background(), cmd, nil, nil))
	require.Equal(t, 2, topology.validators["node1"].heartbeatTracker.blockIndex)
	require.Equal(t, false, topology.validators["node1"].heartbeatTracker.blockSigs[1])

	topology.validators["node1"].data.VegaPubKey = "eeee"
	rejectVega := func(message, signature, pubkey []byte) error { return errors.New("unverifiable vega signature") }
	rejectEth := func(message, signature []byte, hexAddress string) error {
		return errors.New("unverifiable eth signature")
	}
	acceptVega := func(message, signature, pubkey []byte) error { return nil }
	acceptEth := func(message, signature []byte, hexAddress string) error { return nil }

	// unverifiable vega signature
	require.Error(t, topology.ProcessValidatorHeartbeat(context.Background(), cmd, rejectVega, nil))
	require.Equal(t, 3, topology.validators["node1"].heartbeatTracker.blockIndex)
	require.Equal(t, false, topology.validators["node1"].heartbeatTracker.blockSigs[2])

	// undecodable eth signature
	cmd.EthereumSignature = &commandspb.Signature{
		Value: "haha",
	}

	require.Error(t, topology.ProcessValidatorHeartbeat(context.Background(), cmd, acceptVega, nil))
	require.Equal(t, 4, topology.validators["node1"].heartbeatTracker.blockIndex)
	require.Equal(t, false, topology.validators["node1"].heartbeatTracker.blockSigs[3])

	// rejected eth signature
	cmd.EthereumSignature.Value = "ffff"
	require.Error(t, topology.ProcessValidatorHeartbeat(context.Background(), cmd, acceptVega, rejectEth))
	require.Equal(t, 5, topology.validators["node1"].heartbeatTracker.blockIndex)
	require.Equal(t, false, topology.validators["node1"].heartbeatTracker.blockSigs[4])

	// accepted eth signature
	require.NoError(t, topology.ProcessValidatorHeartbeat(context.Background(), cmd, acceptVega, acceptEth))
	require.Equal(t, 6, topology.validators["node1"].heartbeatTracker.blockIndex)
	require.Equal(t, true, topology.validators["node1"].heartbeatTracker.blockSigs[5])
}

func TestGetNodeRequiringHB(t *testing.T) {
	top := getHBTestTopology(t)
	now := time.Now()
	top.epochSeq = 1

	tt := &dummyTestTime{}
	top.timeService = tt
	// initialise all to be not require resend for now
	for _, vs := range top.validators {
		vs.heartbeatTracker.expectedNexthashSince = now.Add(500 * time.Second)
	}

	top.validators["node1"].heartbeatTracker.expectedNexthashSince = now
	top.validators["node2"].heartbeatTracker.expectedNexthashSince = now.Add(-200 * time.Second)
	top.validators["node3"].data.FromEpoch = 2
	top.validators["node3"].heartbeatTracker.expectedNexthashSince = now.Add(-300 * time.Second)

	tt.now = now
	res := top.getNodesRequiringHB()
	require.Equal(t, 0, len(res))

	// move time by 801 seconds
	tt.now = now.Add(801 * time.Second)
	res = top.getNodesRequiringHB()
	require.Equal(t, 1, len(res))
	require.Equal(t, "node2", res[0])

	top.epochSeq = 2
	res = top.getNodesRequiringHB()
	require.Equal(t, 2, len(res))
	require.Equal(t, "node2", res[0])
	require.Equal(t, "node3", res[1])

	// move time by 200 seconds
	tt.now = now.Add(1001 * time.Second)
	res = top.getNodesRequiringHB()
	require.Equal(t, 3, len(res))
	require.Equal(t, "node1", res[0])
	require.Equal(t, "node2", res[1])
	require.Equal(t, "node3", res[2])
}

func getHBTestTopology(t *testing.T) *Topology {
	t.Helper()

	topology := &Topology{}
	topology.validators = map[string]*valState{}
	topology.OnEpochLengthUpdate(context.Background(), 100000*time.Second)
	for i := 0; i < 13; i++ {
		index := strconv.Itoa(i)
		topology.validators["node"+index] = &valState{
			data: ValidatorData{
				ID:        "node" + index,
				FromEpoch: 1,
			},
			heartbeatTracker: &validatorHeartbeatTracker{},
		}
	}
	return topology
}

type dummyTestTime struct {
	now time.Time
}

func (tt *dummyTestTime) GetTimeNow() time.Time {
	return tt.now
}
