package internal

import (
	"vega/internal/vegatime"
	"vega/internal/parties"
	"vega/internal/markets"
)

// Config ties together all other application configuration types.
type Config struct {
	Time vegatime.Config
	Parties parties.Config
	Markets markets.Config
}