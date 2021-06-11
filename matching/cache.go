package matching

import "code.vegaprotocol.io/vega/types"

type BookCache struct {
	indicativePrice          cachedUint
	indicativeVolume         cachedUint
	indicativeUncrossingSide cachedSide
}

type cachedUint struct {
	valid bool
	value uint64
}

type cachedSide struct {
	valid bool
	value types.Side
}

func (c *cachedUint) Set(u uint64) {
	c.value = u
	c.valid = true
}

func (c *cachedUint) Invalidate() {
	c.valid = false
}

func (c *cachedUint) Get() (uint64, bool) {
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

func (c *BookCache) SetIndicativePrice(v uint64) {
	c.indicativePrice.Set(v)
}

func (c *BookCache) GetIndicativePrice() (uint64, bool) {
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
