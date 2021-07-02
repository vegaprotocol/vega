package matching

import (
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

type BookCache struct {
	indicativePrice          cachedUint
	indicativeVolume         cachedUint64
	indicativeUncrossingSide cachedSide
}

func NewBookCache() BookCache {
	return BookCache{
		indicativePrice: cachedUint{
			value: num.NewUint(0),
		},
	}
}

type cachedUint struct {
	valid bool
	value *num.Uint
}

type cachedUint64 struct {
	valid bool
	value uint64
}

type cachedSide struct {
	valid bool
	value types.Side
}

func (c *cachedUint) Set(u *num.Uint) {
	c.value = u
	c.valid = true
}

func (c *cachedUint) Invalidate() {
	c.valid = false
}

func (c *cachedUint) Get() (*num.Uint, bool) {
	return c.value.Clone(), c.valid
}

func (c *cachedUint64) Set(u uint64) {
	c.value = u
	c.valid = true
}

func (c *cachedUint64) Invalidate() {
	c.valid = false
}

func (c *cachedUint64) Get() (uint64, bool) {
	return c.value, c.valid
}

func (c *cachedSide) Set(s types.Side) {
	c.value = s
	c.valid = true
}

func (c *cachedSide) Invalidate() {
	c.valid = false
}

func (c *cachedSide) Get() (types.Side, bool) {
	return c.value, c.valid
}

func (c *BookCache) Invalidate() {
	c.indicativePrice.Invalidate()
	c.indicativeVolume.Invalidate()
	c.indicativeUncrossingSide.Invalidate()
}

func (c *BookCache) SetIndicativePrice(v *num.Uint) {
	c.indicativePrice.Set(v)
}

func (c *BookCache) GetIndicativePrice() (*num.Uint, bool) {
	return c.indicativePrice.Get()
}

func (c *BookCache) SetIndicativeVolume(v uint64) {
	c.indicativeVolume.Set(v)
}

func (c *BookCache) GetIndicativeVolume() (uint64, bool) {
	return c.indicativeVolume.Get()
}

func (c *BookCache) SetIndicativeUncrossingSide(s types.Side) {
	c.indicativeUncrossingSide.Set(s)
}

func (c *BookCache) GetIndicativeUncrossingSide() (types.Side, bool) {
	return c.indicativeUncrossingSide.Get()
}
