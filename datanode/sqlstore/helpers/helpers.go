package helpers

import (
	"code.vegaprotocol.io/vega/datanode/entities"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
)

// GenerateID generates a 256 bit pseudo-random hash ID.
func GenerateID() string {
	return vgcrypto.RandomHash()
}

func DefaultNoPagination() entities.CursorPagination {
	pagination, _ := entities.NewCursorPagination(nil, nil, nil, nil, true)
	return pagination
}
