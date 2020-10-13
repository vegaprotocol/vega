package settlement

import (
	"errors"
	"strings"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
)

// namedLogger is the identifier for package and should ideally match the package name
// this is simply emitted as a hierarchical label e.g. 'api.grpc'.
const namedLogger = "settlement"

var (
	ErrInvalidFinalSettlement = errors.New("invalid final settlement")
)

type FinalSettlement string

const (
	FinalSettlementOracle    FinalSettlement = "Oracle"
	FinalSettlementMarkPrice                 = "LastMarkPrice"
)

type FinalSettlementW struct {
	FinalSettlement
}

func (f *FinalSettlementW) Get() FinalSettlement {
	return f.FinalSettlement
}

func (f *FinalSettlementW) UnmarshalText(text []byte) error {
	var err error
	switch strings.ToLower(string(text)) {
	case strings.ToLower(string(FinalSettlementMarkPrice)):
		f.FinalSettlement = FinalSettlementMarkPrice
	case strings.ToLower(string(FinalSettlementOracle)):
		f.FinalSettlement = FinalSettlementOracle
	default:
		err = ErrInvalidFinalSettlement
	}
	return err
}

func (f *FinalSettlementW) UnmarshalFlag(text string) error {
	return f.UnmarshalText([]byte(text))
}

func (f FinalSettlementW) MarshalText() ([]byte, error) {
	return []byte(f.FinalSettlement), nil
}

// Config represent the configuration of the settlement engine
type Config struct {
	Level           encoding.LogLevel `long:"log-level"`
	FinalSettlement FinalSettlementW  `long:"final-settlement"`
}

// NewDefaultConfig creates an instance of the package specific configuration, given a
// pointer to a logger instance to be used for logging within the package.
func NewDefaultConfig() Config {
	return Config{
		Level:           encoding.LogLevel{Level: logging.InfoLevel},
		FinalSettlement: FinalSettlementW{FinalSettlement: FinalSettlementOracle},
	}
}
