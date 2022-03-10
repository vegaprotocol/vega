package entities

import (
	"encoding/hex"
	"fmt"
	"time"
)

type Party struct {
	ID       []byte
	VegaTime time.Time
}

func (p Party) HexID() string {
	if len(p.ID) == 0 {
		return "network"
	}
	return hex.EncodeToString(p.ID)
}

func MakePartyID(stringID string) ([]byte, error) {
	if stringID == "network" {
		return []byte{}, nil
	}
	id, err := hex.DecodeString(stringID)
	if err != nil {
		return nil, fmt.Errorf("party id is not valid hex string: %v", stringID)
	}
	return id, nil
}
