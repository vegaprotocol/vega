package helpers

import (
	"math/rand"
	"time"
)

type Block struct {
	Duration int64
	Variance float64
}

func NewBlock() *Block {
	return &Block{
		Duration: 1,
	}
}

func (b Block) GetDuration() time.Duration {
	if b.Variance == 0 {
		return time.Duration(b.Duration) * time.Second
	}
	base := time.Duration(b.Duration) * time.Second
	factor := int64(b.Variance * float64(time.Second))
	// factor of 3, random 0-6 yields random number between -3 and +3
	offset := factor - rand.Int63n(2*factor)
	return base + time.Duration(offset)
}

func (b Block) GetStep(d time.Duration) time.Duration {
	if b.Variance == 0 {
		return d
	}
	factor := int64(b.Variance * float64(d))
	offset := factor - rand.Int63n(factor*2)
	return d + time.Duration(offset)
}
