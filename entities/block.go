package entities

import (
	"time"
)

type Block struct {
	VegaTime time.Time
	Height   int64
	Hash     []byte
}
