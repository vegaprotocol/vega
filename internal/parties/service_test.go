package parties_test

import (
	"context"
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/internal/auth"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/parties"
	"code.vegaprotocol.io/vega/internal/parties/mocks"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testService struct {
	*parties.Svc
	log   *logging.Logger
	ctx   context.Context
	cfunc context.CancelFunc
	ctrl  *gomock.Controller
	store *mocks.MockPartyStore
}

func getTestService(t *testing.T) *testService {
	ctrl := gomock.NewController(t)
	store := mocks.NewMockPartyStore(ctrl)
	log := logging.NewTestLogger()
	ctx, cfunc := context.WithCancel(context.Background())
	svc, err := parties.NewService(
		log,
		parties.NewDefaultConfig(),
		store,
	)
	assert.NoError(t, err)
	return &testService{
		Svc:   svc,
		log:   log,
		ctx:   ctx,
		cfunc: cfunc,
		ctrl:  ctrl,
		store: store,
	}
}

func TestPartyService_CreateParty(t *testing.T) {
	svc := getTestService(t)
	defer svc.Finish()
	p := &types.Party{Id: "Christina"}

	svc.store.EXPECT().Post(p).Times(1).Return(nil)

	assert.NoError(t, svc.CreateParty(svc.ctx, p))
}

func TestPartyService_GetAll(t *testing.T) {
	svc := getTestService(t)
	defer svc.Finish()

	expected := []*types.Party{
		{Id: "Edd"},
		{Id: "Barney"},
		{Id: "Ramsey"},
		{Id: "Jeremy"},
	}

	svc.store.EXPECT().GetAll().Times(1).Return(expected, nil)

	parties, err := svc.GetAll(svc.ctx)

	assert.NoError(t, err)
	assert.Equal(t, expected, parties)
}

func TestPartyService_GetByName(t *testing.T) {
	svc := getTestService(t)
	defer svc.Finish()

	expect := &types.Party{
		Id: "Candida",
	}
	svc.store.EXPECT().GetByID(expect.Id).Times(1).Return(expect, nil)

	party, err := svc.GetByID(svc.ctx, expect.Id)
	assert.NoError(t, err)
	assert.Equal(t, expect, party)
}

func TestPartyService_UpdateParties(t *testing.T) {
	svc := getTestService(t)
	defer svc.Finish()

	const pid01 = "pid01"

	ps := []auth.PartyInfo{{ID: pid01}}
	svc.store.EXPECT().GetByID(pid01).Times(1).Return(nil, errors.New("party not found"))
	svc.store.EXPECT().Post(&types.Party{Id: pid01}).Times(1).Return(nil)
	svc.UpdateParties(ps)

	svc.store.EXPECT().GetByID(pid01).Times(1).Return(&types.Party{Id: pid01}, nil)
	svc.UpdateParties(ps)
}

func (t *testService) Finish() {
	t.log.Sync()
	t.cfunc()
	t.ctrl.Finish()
}
