package close

type Closer struct {
	closeFns []func()
}

// Add adds a function to call during call to CloseAll.
func (c *Closer) Add(closeFn func()) {
	c.closeFns = append(c.closeFns, closeFn)
}

// CloseAll calls all close functions in reverse order.
// Higher level-components should be closed first, but are usually instantiated
// last (and, thus, added later to the closer), hence the reverse order.
func (c *Closer) CloseAll() {
	for i := len(c.closeFns) - 1; i >= 0; i-- {
		c.closeFns[i]()
	}

	c.closeFns = []func(){}
}

func NewCloser() *Closer {
	return &Closer{
		closeFns: []func(){},
	}
}
