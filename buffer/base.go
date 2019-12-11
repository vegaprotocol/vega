package buffer

type base struct {
	flush chan struct{}
	unsub chan int
	keys  []int
	subs  map[int]struct{}
}

func newBase() base {
	return base{
		flush: make(chan struct{}),
		unsub: make(chan int),
		keys:  []int{},
		subs:  map[int]struct{}{},
	}
}

func (b *base) Flush() {
	b.flush <- struct{}{}
}

func (b *base) getKey() int {
	if len(b.keys) == 0 {
		return len(b.subs) + 1
	}
	k := b.keys[0]
	b.keys = b.keys[1:]
	return k
}

func (b *base) subscribe(k int) {
	b.subs[k] = struct{}{}
}

func (b *base) unsubscribe(u int) {
	if _, ok := b.subs[u]; ok {
		b.keys = append(b.keys, u)
	}
	delete(b.subs, u)
}

func (b *base) done() {
	close(b.flush)
	close(b.unsub)
}
