package plugins

import (
	"context"
	"sort"
	"sync"

	"code.vegaprotocol.io/vega/buffer"
	"code.vegaprotocol.io/vega/logging"
	"google.golang.org/grpc"
)

var (
	pluginsMu sync.RWMutex
	plugins   = make(map[string]Plugin)
)

type Plugin interface {
	New(*logging.Logger, context.Context, *buffer.Buffers, *grpc.Server) Plugin
	Start() error
}

// Register makes a plugin available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, plugin Plugin) {
	pluginsMu.Lock()
	defer pluginsMu.Unlock()
	if plugin == nil {
		panic("plugins: Register plugin is nil")
	}

	if _, dup := plugins[name]; dup {
		panic("plugins: Register called twice for plugin " + name)
	}
	plugins[name] = plugin
}

func unregisterAllPlugins() {
	pluginsMu.Lock()
	defer pluginsMu.Unlock()
	// For tests.
	plugins = make(map[string]Plugin)
}

// Plugins returns a sorted list of the names of the registered plugins.
func Plugins() []string {
	pluginsMu.RLock()
	defer pluginsMu.RUnlock()
	var list []string
	for name := range plugins {
		list = append(list, name)
	}
	sort.Strings(list)
	return list

}

func Get(name string) (Plugin, bool) {
	pluginsMu.RLock()
	defer pluginsMu.RUnlock()
	p, ok := plugins[name]
	return p, ok
}
