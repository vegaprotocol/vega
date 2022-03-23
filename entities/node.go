package entities

import (
	"encoding/hex"
	"fmt"
	"time"
)

type Node struct {
	ID       []byte
	VegaTime time.Time
}

func (n Node) HexID() string {
	return hex.EncodeToString(n.ID)
}

func MakeNodeID(stringID string) ([]byte, error) {
	id, err := hex.DecodeString(stringID)
	if err != nil {
		return nil, fmt.Errorf("node id is not valid hex string: %v", stringID)
	}
	return id, nil
}
