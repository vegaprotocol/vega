package basecmd

import (
	"os"
	"sync"
)

var (
	mu          sync.Mutex
	atExitFuncs []func()
)

func init() {
	atExitFuncs = []func(){}
}

func AtExit(f func()) {
	mu.Lock()
	atExitFuncs = append(atExitFuncs, f)
	mu.Unlock()
}

func Exit(status int) {
	mu.Lock()
	for _, f := range atExitFuncs {
		f()
	}
	os.Exit(status)
}
