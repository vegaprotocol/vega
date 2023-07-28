package test

import (
	"sync"
)

func OnlyOnce(f func()) func() {
	var once sync.Once

	return func() {
		once.Do(f)
	}
}
