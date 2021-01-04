package parties_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/parties"
	"code.vegaprotocol.io/vega/parties/mocks"
	types "code.vegaprotocol.io/vega/proto/gen/golang"

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
	store.EXPECT().Post(gomock.Any()).Times(1).Return(nil)
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

func (t *testService) Finish() {
	t.log.Sync()
	t.cfunc()
	t.ctrl.Finish()
}
