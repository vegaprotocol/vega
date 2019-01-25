package parties

import "vega/msg"

type Party struct {
	Name string
	Positions []msg.MarketPosition
}