package internal

import (
	"vega/internal/vegatime"
	"vega/internal/parties"
)

// Config ties together all other application configuration types.
type Config struct {
	Time vegatime.Config
	Parties parties.Config
}