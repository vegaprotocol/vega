package risk_test

import (
	"testing"

	"code.vegaprotocol.io/vega/internal/risk"
	"github.com/golang/mock/gomock"
)

type testEngine struct {
	*risk.Engine
	ctrl *gomock.Controller
}

func TestMargin(t *testing.T) {
}

func getTestEngine(t *testing.T) {
	ctrl := gomock.NewController(t)
}
