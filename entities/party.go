package entities

import (
	"encoding/hex"
	"time"
)

type Party struct {
	ID       []byte
	VegaTime time.Time
}

func (p Party) HexId() string {
	return hex.EncodeToString(p.ID)
}
