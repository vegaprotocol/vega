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

package validators_test

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/core/validators/mocks"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testWitness struct {
	*validators.Witness
	ctrl      *gomock.Controller
	top       *mocks.MockValidatorTopology
	cmd       *mocks.MockCommander
	tsvc      *mocks.MockTimeService
	startTime time.Time
}

func getTestWitness(t *testing.T) *testWitness {
	t.Helper()
	ctrl := gomock.NewController(t)
	top := mocks.NewMockValidatorTopology(ctrl)
	cmd := mocks.NewMockCommander(ctrl)
	tsvc := mocks.NewMockTimeService(ctrl)

	now := time.Now()
	tsvc.EXPECT().GetTimeNow().Times(1).Return(now)
	w := validators.NewWitness(context.Background(),
		logging.NewTestLogger(), validators.NewDefaultConfig(), top, cmd, tsvc)
	assert.NotNil(t, w)

	return &testWitness{
		Witness:   w,
		ctrl:      ctrl,
		top:       top,
		cmd:       cmd,
		tsvc:      tsvc,
		startTime: now,
	}
}

func TestExtResCheck(t *testing.T) {
	t.Run("start - error duplicate", testStartErrorDuplicate)
	t.Run("start - error check failed", testStartErrorCheckFailed)
	t.Run("start - OK", testStartOK)
	t.Run("add node vote - error invalid id", testNodeVoteInvalidProposalReference)
	t.Run("add node vote - error note a validator", testNodeVoteNotAValidator)
	t.Run("add node vote - error duplicate vote", testNodeVoteDuplicateVote)
	t.Run("add node vote - OK", testNodeVoteOK)
	t.Run("on chain time update validated asset", testOnTick)
	t.Run("on chain time update validated asset - non validator", testOnTickNonValidator)
}

func testStartErrorDuplicate(t *testing.T) {
	erc := getTestWitness(t)
	defer erc.ctrl.Finish()
	defer erc.Stop()

	erc.top.EXPECT().IsValidator().AnyTimes().Return(true)
	res := testRes{"resource-id-1", func() error {
		return nil
	}}
	checkUntil := erc.startTime.Add(700 * time.Second)
	cb := func(interface{}, bool) {}

	err := erc.StartCheck(res, cb, checkUntil)
	assert.NoError(t, err)
	err = erc.StartCheck(res, cb, checkUntil)
	assert.EqualError(t, err, validators.ErrResourceDuplicate.Error())
}

func testStartErrorCheckFailed(t *testing.T) {
	erc := getTestWitness(t)
	defer erc.ctrl.Finish()
	defer erc.Stop()

	erc.top.EXPECT().IsValidator().AnyTimes().Return(true)
	res := testRes{"resource-id-1", func() error {
		return nil
	}}
	checkUntil := erc.startTime.Add(31 * 24 * time.Hour)
	cb := func(interface{}, bool) {}

	err := erc.StartCheck(res, cb, checkUntil)
	assert.EqualError(t, err, validators.ErrCheckUntilInvalid.Error())
}

func testStartOK(t *testing.T) {
	erc := getTestWitness(t)
	defer erc.ctrl.Finish()
	defer erc.Stop()

	erc.top.EXPECT().IsValidator().AnyTimes().Return(true)
	res := testRes{"resource-id-1", func() error {
		return nil
	}}
	checkUntil := erc.startTime.Add(700 * time.Second)
	cb := func(interface{}, bool) {}

	err := erc.StartCheck(res, cb, checkUntil)
	assert.NoError(t, err)
}

func testNodeVoteInvalidProposalReference(t *testing.T) {
	erc := getTestWitness(t)
	defer erc.ctrl.Finish()
	defer erc.Stop()

	erc.top.EXPECT().IsValidator().AnyTimes().Return(true)
	res := testRes{"resource-id-1", func() error {
		return nil
	}}
	checkUntil := erc.startTime.Add(700 * time.Second)
	cb := func(interface{}, bool) {}

	err := erc.StartCheck(res, cb, checkUntil)
	assert.NoError(t, err)

	pubKey := newPublicKey("somepubkey")
	err = erc.AddNodeCheck(context.Background(), &commandspb.NodeVote{Reference: "bad-id"}, pubKey)
	assert.EqualError(t, err, validators.ErrInvalidResourceIDForNodeVote.Error())
}

func testNodeVoteNotAValidator(t *testing.T) {
	erc := getTestWitness(t)
	defer erc.ctrl.Finish()
	defer erc.Stop()

	erc.top.EXPECT().IsValidator().AnyTimes().Return(true)
	res := testRes{"resource-id-1", func() error {
		return nil
	}}
	checkUntil := erc.startTime.Add(700 * time.Second)
	cb := func(interface{}, bool) {}

	err := erc.StartCheck(res, cb, checkUntil)
	assert.NoError(t, err)

	pubKey := newPublicKey("somepubkey")
	erc.top.EXPECT().IsValidatorVegaPubKey(pubKey.Hex()).Times(1).Return(false)
	err = erc.AddNodeCheck(context.Background(), &commandspb.NodeVote{Reference: res.id}, pubKey)
	assert.EqualError(t, err, validators.ErrVoteFromNonValidator.Error())
}

func testNodeVoteOK(t *testing.T) {
	erc := getTestWitness(t)
	defer erc.ctrl.Finish()
	defer erc.Stop()

	erc.top.EXPECT().IsValidator().AnyTimes().Return(true)

	res := testRes{"resource-id-1", func() error {
		return nil
	}}
	checkUntil := erc.startTime.Add(700 * time.Second)
	cb := func(interface{}, bool) {}

	err := erc.StartCheck(res, cb, checkUntil)
	assert.NoError(t, err)

	pubKey := newPublicKey("somepubkey")
	erc.top.EXPECT().IsValidatorVegaPubKey(pubKey.Hex()).Times(1).Return(true)
	err = erc.AddNodeCheck(context.Background(), &commandspb.NodeVote{Reference: res.id}, pubKey)
	assert.NoError(t, err)
}

func testNodeVoteDuplicateVote(t *testing.T) {
	erc := getTestWitness(t)
	defer erc.ctrl.Finish()
	defer erc.Stop()

	erc.top.EXPECT().IsValidator().AnyTimes().Return(true)
	res := testRes{"resource-id-1", func() error {
		return nil
	}}
	checkUntil := erc.startTime.Add(700 * time.Second)
	cb := func(interface{}, bool) {}

	err := erc.StartCheck(res, cb, checkUntil)
	assert.NoError(t, err)

	// first vote, all good
	pubKey := newPublicKey("somepubkey")
	erc.top.EXPECT().IsValidatorVegaPubKey(pubKey.Hex()).Times(1).Return(true)
	err = erc.AddNodeCheck(context.Background(), &commandspb.NodeVote{Reference: res.id}, pubKey)
	require.NoError(t, err)

	// second vote, bad
	erc.top.EXPECT().IsValidatorVegaPubKey(pubKey.Hex()).Times(1).Return(true)
	err = erc.AddNodeCheck(context.Background(), &commandspb.NodeVote{Reference: res.id}, pubKey)
	require.EqualError(t, err, validators.ErrDuplicateVoteFromNode.Error())
}

func TestVoteMajorityCalculation(t *testing.T) {
	erc := getTestWitness(t)
	defer erc.ctrl.Finish()
	defer erc.Stop()
	erc.Witness.OnDefaultValidatorsVoteRequiredUpdate(context.Background(), num.DecimalFromFloat(0.67))
	selfPubKey := "b7ee437dc100d642"

	erc.top.EXPECT().GetTotalVotingPower().AnyTimes().Return(int64(500))
	erc.top.EXPECT().GetVotingPower(gomock.Any()).AnyTimes().Return(int64(100))
	erc.top.EXPECT().IsValidator().AnyTimes().Return(true)

	ch := make(chan struct{}, 1)
	res := testRes{"resource-id-1", func() error {
		ch <- struct{}{}
		return nil
	}}
	checkUntil := erc.startTime.Add(700 * time.Second)
	cb := func(interface{}, bool) {
		// unblock chanel listen to finish test
		ch <- struct{}{}
	}

	err := erc.StartCheck(res, cb, checkUntil)
	assert.NoError(t, err)

	// first wait once for the asset to be validated
	<-ch

	// first on chain time update, we send our own vote
	erc.cmd.EXPECT().Command(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	newNow := erc.startTime.Add(1 * time.Second)
	erc.OnTick(context.Background(), newNow)

	// then we propagate our own vote
	pubKey := newPublicKey(selfPubKey)
	erc.top.EXPECT().IsValidatorVegaPubKey(pubKey.Hex()).Times(1).Return(true)
	err = erc.AddNodeCheck(context.Background(), &commandspb.NodeVote{Reference: res.id}, pubKey)
	assert.NoError(t, err)

	for i := 0; i < 3; i++ {
		// second vote from another validator
		othPubKey := newPublicKey(crypto.RandomHash())
		erc.top.EXPECT().IsValidatorVegaPubKey(othPubKey.Hex()).Times(1).Return(true)
		err = erc.AddNodeCheck(context.Background(), &commandspb.NodeVote{Reference: res.id}, othPubKey)
		assert.NoError(t, err)
	}

	// we have 4 votes, that should suffice to pass
	// call onTick again to get the callback called
	newNow = newNow.Add(1 * time.Second)
	erc.top.EXPECT().IsTendermintValidator(gomock.Any()).Times(4).Return(true)
	erc.OnTick(context.Background(), newNow)

	// block to wait for the result
	<-ch
}

func testOnTick(t *testing.T) {
	erc := getTestWitness(t)
	defer erc.ctrl.Finish()
	defer erc.Stop()

	selfPubKey := "b7ee437dc100d642"

	erc.Witness.OnDefaultValidatorsVoteRequiredUpdate(context.Background(), num.DecimalFromFloat(0.67))
	erc.top.EXPECT().GetTotalVotingPower().AnyTimes().Return(int64(298))
	erc.top.EXPECT().GetVotingPower(gomock.Any()).AnyTimes().Return(int64(100))
	erc.top.EXPECT().IsValidator().AnyTimes().Return(true)

	ch := make(chan struct{}, 1)
	res := testRes{"resource-id-1", func() error {
		ch <- struct{}{}
		return nil
	}}
	checkUntil := erc.startTime.Add(700 * time.Second)
	cb := func(interface{}, bool) {
		// unblock chanel listen to finish test
		ch <- struct{}{}
	}

	err := erc.StartCheck(res, cb, checkUntil)
	assert.NoError(t, err)

	// first wait once for the asset to be validated
	<-ch

	// first on chain time update, we send our own vote
	erc.cmd.EXPECT().Command(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	newNow := erc.startTime.Add(1 * time.Second)
	erc.OnTick(context.Background(), newNow)

	// then we propagate our own vote
	pubKey := newPublicKey(selfPubKey)
	erc.top.EXPECT().IsValidatorVegaPubKey(pubKey.Hex()).Times(1).Return(true)
	err = erc.AddNodeCheck(context.Background(), &commandspb.NodeVote{Reference: res.id}, pubKey)
	assert.NoError(t, err)

	// second vote from another validator
	othPubKey := newPublicKey("somepubkey")
	erc.top.EXPECT().IsValidatorVegaPubKey(othPubKey.Hex()).Times(1).Return(true)
	err = erc.AddNodeCheck(context.Background(), &commandspb.NodeVote{Reference: res.id}, othPubKey)
	assert.NoError(t, err)

	// call onTick again to get the callback called
	newNow = newNow.Add(1 * time.Second)
	erc.top.EXPECT().IsTendermintValidator(gomock.Any()).Times(2).Return(true)
	erc.OnTick(context.Background(), newNow)

	// block to wait for the result
	<-ch
}

func testOnTickNonValidator(t *testing.T) {
	erc := getTestWitness(t)
	defer erc.ctrl.Finish()
	defer erc.Stop()

	selfPubKey := "b7ee437dc100d642"

	erc.top.EXPECT().GetTotalVotingPower().AnyTimes().Return(int64(298))
	erc.top.EXPECT().GetVotingPower(gomock.Any()).AnyTimes().Return(int64(100))
	erc.top.EXPECT().IsValidator().AnyTimes().Return(false)

	res := testRes{"resource-id-1", func() error {
		return nil
	}}

	checkUntil := erc.startTime.Add(700 * time.Second)
	cb := func(interface{}, bool) {}

	err := erc.StartCheck(res, cb, checkUntil)
	assert.NoError(t, err)

	// first on chain time update, we send our own vote
	erc.cmd.EXPECT().Command(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	newNow := erc.startTime.Add(1 * time.Second)
	erc.OnTick(context.Background(), newNow)

	// then we propagate our own vote
	pubKey := newPublicKey(selfPubKey)
	erc.top.EXPECT().IsValidatorVegaPubKey(pubKey.Hex()).Times(1).Return(true)
	err = erc.AddNodeCheck(context.Background(), &commandspb.NodeVote{Reference: res.id}, pubKey)
	assert.NoError(t, err)

	// second vote from another validator
	othPubKey := newPublicKey("somepubkey")
	erc.top.EXPECT().IsValidatorVegaPubKey(othPubKey.Hex()).Times(1).Return(true)
	err = erc.AddNodeCheck(context.Background(), &commandspb.NodeVote{Reference: res.id}, othPubKey)
	assert.NoError(t, err)

	// call onTick again to get the callback called
	newNow = newNow.Add(1 * time.Second)
	erc.top.EXPECT().IsTendermintValidator(gomock.Any()).Times(2).Return(true)
	erc.OnTick(context.Background(), newNow)
}

type testRes struct {
	id    string
	check func() error
}

func (t testRes) GetID() string { return t.id }
func (t testRes) GetType() commandspb.NodeVote_Type {
	return commandspb.NodeVote_TYPE_FUNDS_DEPOSITED
}
func (t testRes) Check(ctx context.Context) error { return t.check() }

func newPublicKey(k string) crypto.PublicKey {
	pubKeyB := []byte(k)
	return crypto.NewPublicKey(hex.EncodeToString(pubKeyB), pubKeyB)
}
