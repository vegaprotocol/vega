package entities

import (
	"encoding/hex"
	"fmt"
	"time"
)

type Market struct {
	ID       []byte
	VegaTime time.Time
}

func MakeMarketID(stringID string) ([]byte, error) {
	id, err := hex.DecodeString(stringID)
	if err != nil {
		return nil, fmt.Errorf("market id is not valid hex string: %v", stringID)
	}
	return id, nil
}

func (m Market) HexID() string {
	return hex.EncodeToString(m.ID)
}
