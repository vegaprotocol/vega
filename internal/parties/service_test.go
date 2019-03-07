package parties

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage/mocks"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/stretchr/testify/assert"
)

func TestPartyService_NewService(t *testing.T) {
	partyStore := &mocks.PartyStore{}

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	partyConfig := NewDefaultConfig(logger)
	partyService, err := NewPartyService(partyConfig, partyStore)
	assert.NotNil(t, partyService)
	assert.Nil(t, err)
}

func TestPartyService_CreateParty(t *testing.T) {
	p := &types.Party{Name: "Christina"}

	partyStore := &mocks.PartyStore{}
	partyStore.On("Post", p).Return(nil)

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	partyConfig := NewDefaultConfig(logger)
	partyService, err := NewPartyService(partyConfig, partyStore)
	assert.NotNil(t, partyService)
	assert.Nil(t, err)

	err = partyService.CreateParty(context.Background(), p)
	assert.Nil(t, err)
}

func TestPartyService_GetAll(t *testing.T) {
	partyStore := &mocks.PartyStore{}

	partyStore.On("GetAll").Return([]*types.Party{
		{Name: "Edd"},
		{Name: "Barney"},
		{Name: "Ramsey"},
		{Name: "Jeremy"},
	}, nil).Once()

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	partyConfig := NewDefaultConfig(logger)
	partyService, err := NewPartyService(partyConfig, partyStore)
	assert.NotNil(t, partyService)
	assert.Nil(t, err)

	parties, err := partyService.GetAll(context.Background())

	assert.Len(t, parties, 4)
	assert.Equal(t, "Edd", parties[0].Name)
	assert.Equal(t, "Barney", parties[1].Name)
	assert.Equal(t, "Ramsey", parties[2].Name)
	assert.Equal(t, "Jeremy", parties[3].Name)
}

func TestPartyService_GetByName(t *testing.T) {
	partyStore := &mocks.PartyStore{}

	partyStore.On("GetByName", "Candida").Return(&types.Party{
		Name: "Candida",
	}, nil).Once()

	logger := logging.NewLoggerFromEnv("dev")
	defer logger.Sync()

	partyConfig := NewDefaultConfig(logger)
	partyService, err := NewPartyService(partyConfig, partyStore)
	assert.NotNil(t, partyService)
	assert.Nil(t, err)

	party, err := partyService.GetByName(context.Background(), "Candida")
	assert.Nil(t, err)

	assert.Equal(t, "Candida", party.Name)

}
