package validators_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/validators"
	"code.vegaprotocol.io/vega/validators/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testExtResChecker struct {
	*validators.ExtResChecker
	ctrl      *gomock.Controller
	top       *mocks.MockValidatorTopology
	cmd       *mocks.MockCommander
	tsvc      *mocks.MockTimeService
	startTime time.Time
}

func getTestExtResChecker(t *testing.T) *testExtResChecker {
	ctrl := gomock.NewController(t)
	top := mocks.NewMockValidatorTopology(ctrl)
	cmd := mocks.NewMockCommander(ctrl)
	tsvc := mocks.NewMockTimeService(ctrl)

	now := time.Now()
	tsvc.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	tsvc.EXPECT().NotifyOnTick(gomock.Any()).Times(1)
	nv := validators.NewExtResChecker(
		logging.NewTestLogger(), top, cmd, tsvc)
	assert.NotNil(t, nv)

	return &testExtResChecker{
		ExtResChecker: nv,
		ctrl:          ctrl,
		top:           top,
		cmd:           cmd,
		tsvc:          tsvc,
		startTime:     now,
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
	t.Run("on chain time update validated asset", testOnChainTimeUpdate)
}

func testStartErrorDuplicate(t *testing.T) {
	erc := getTestExtResChecker(t)
	defer erc.ctrl.Finish()
	defer erc.Stop()

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
	erc := getTestExtResChecker(t)
	defer erc.ctrl.Finish()
	defer erc.Stop()

	res := testRes{"resource-id-1", func() error {
		return nil
	}}
	checkUntil := erc.startTime.Add(1 * time.Second)
	cb := func(interface{}, bool) {}

	err := erc.StartCheck(res, cb, checkUntil)
	assert.EqualError(t, err, validators.ErrCheckUntilInvalid.Error())
}

func testStartOK(t *testing.T) {
	erc := getTestExtResChecker(t)
	defer erc.ctrl.Finish()
	defer erc.Stop()

	res := testRes{"resource-id-1", func() error {
		return nil
	}}
	checkUntil := erc.startTime.Add(700 * time.Second)
	cb := func(interface{}, bool) {}

	err := erc.StartCheck(res, cb, checkUntil)
	assert.NoError(t, err)
}

func testNodeVoteInvalidProposalReference(t *testing.T) {
	erc := getTestExtResChecker(t)
	defer erc.ctrl.Finish()
	defer erc.Stop()

	res := testRes{"resource-id-1", func() error {
		return nil
	}}
	checkUntil := erc.startTime.Add(700 * time.Second)
	cb := func(interface{}, bool) {}

	err := erc.StartCheck(res, cb, checkUntil)
	assert.NoError(t, err)

	err = erc.AddNodeCheck(&proto.NodeVote{Reference: "bad-id"})
	assert.EqualError(t, err, validators.ErrInvalidResourceIDForNodeVote.Error())
}

func testNodeVoteNotAValidator(t *testing.T) {
	erc := getTestExtResChecker(t)
	defer erc.ctrl.Finish()
	defer erc.Stop()

	res := testRes{"resource-id-1", func() error {
		return nil
	}}
	checkUntil := erc.startTime.Add(700 * time.Second)
	cb := func(interface{}, bool) {}

	err := erc.StartCheck(res, cb, checkUntil)
	assert.NoError(t, err)

	erc.top.EXPECT().Exists(gomock.Any()).Times(1).Return(false)
	err = erc.AddNodeCheck(&proto.NodeVote{Reference: res.id})
	assert.EqualError(t, err, validators.ErrVoteFromNonValidator.Error())
}

func testNodeVoteOK(t *testing.T) {
	erc := getTestExtResChecker(t)
	defer erc.ctrl.Finish()
	defer erc.Stop()

	res := testRes{"resource-id-1", func() error {
		return nil
	}}
	checkUntil := erc.startTime.Add(700 * time.Second)
	cb := func(interface{}, bool) {}

	err := erc.StartCheck(res, cb, checkUntil)
	assert.NoError(t, err)

	erc.top.EXPECT().Exists(gomock.Any()).Times(1).Return(true)
	err = erc.AddNodeCheck(&proto.NodeVote{Reference: res.id})
	assert.NoError(t, err)
}

func testNodeVoteDuplicateVote(t *testing.T) {
	erc := getTestExtResChecker(t)
	defer erc.ctrl.Finish()
	defer erc.Stop()

	res := testRes{"resource-id-1", func() error {
		return nil
	}}
	checkUntil := erc.startTime.Add(700 * time.Second)
	cb := func(interface{}, bool) {}

	err := erc.StartCheck(res, cb, checkUntil)
	assert.NoError(t, err)

	// first vote, all good
	erc.top.EXPECT().Exists(gomock.Any()).Times(1).Return(true)
	err = erc.AddNodeCheck(&proto.NodeVote{Reference: res.id, PubKey: []byte("somepubkey")})
	assert.NoError(t, err)

	// second vote, bad
	erc.top.EXPECT().Exists(gomock.Any()).Times(1).Return(true)
	err = erc.AddNodeCheck(&proto.NodeVote{Reference: res.id, PubKey: []byte("somepubkey")})
	assert.EqualError(t, err, validators.ErrDuplicateVoteFromNode.Error())
}

func testOnChainTimeUpdate(t *testing.T) {
	erc := getTestExtResChecker(t)
	defer erc.ctrl.Finish()
	defer erc.Stop()

	selfPubKey := []byte("selfPubKey")

	erc.top.EXPECT().Len().AnyTimes().Return(2)
	erc.top.EXPECT().IsValidator().AnyTimes().Return(true)
	erc.top.EXPECT().SelfVegaPubKey().AnyTimes().Return(selfPubKey)

	ch := make(chan struct{}, 1)
	res := testRes{"resource-id-1", func() error {
		return nil
	}}
	checkUntil := erc.startTime.Add(700 * time.Second)
	cb := func(interface{}, bool) {
		// unblock chanel listen to finish test
		ch <- struct{}{}
	}

	err := erc.StartCheck(res, cb, checkUntil)
	assert.NoError(t, err)

	// first on chain time update, we send our own vote
	erc.cmd.EXPECT().Command(gomock.Any(), gomock.Any()).Times(1).Return(nil)
	newNow := erc.startTime.Add(1 * time.Second)
	erc.OnTick(newNow)

	// then we propagate our own vote
	erc.top.EXPECT().Exists(gomock.Any()).Times(1).Return(true)
	err = erc.AddNodeCheck(&proto.NodeVote{Reference: res.id, PubKey: selfPubKey})
	assert.NoError(t, err)

	// second vote from another validator
	erc.top.EXPECT().Exists(gomock.Any()).Times(1).Return(true)
	err = erc.AddNodeCheck(&proto.NodeVote{Reference: res.id, PubKey: []byte("somepubkey")})
	assert.NoError(t, err)

	// call onTick again to get the callback called
	newNow = newNow.Add(1 * time.Second)
	erc.OnTick(newNow)

	// block to wait for the result
	<-ch
}

type testRes struct {
	id    string
	check func() error
}

func (t testRes) GetID() string { return t.id }
func (t testRes) Check() error  { return t.check() }
