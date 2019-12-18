package plugins

import (
	"context"
	"sort"
	"sync"

	"code.vegaprotocol.io/vega/buffer"
	"code.vegaprotocol.io/vega/logging"

	"github.com/mohae/deepcopy"
	"google.golang.org/grpc"
)

var (
	pluginsMu sync.RWMutex
	plugins   = make(map[string]Plugin)

	configsMu sync.RWMutex
	configs   = make(map[string]interface{})
)

type Buffers interface {
	TradesSub(buf int) buffer.TradeSub
	OrdersSub(buf int) buffer.OrderSub
	MarketsSub(buf int) buffer.MarketSub
}

type Plugin interface {
	New(*logging.Logger, context.Context, Buffers, *grpc.Server, interface{}) (Plugin, error)
	Start() error
}

// Register makes a plugin available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, plugin Plugin, cfg interface{}) {
	pluginsMu.Lock()
	defer pluginsMu.Unlock()
	if plugin == nil {
		panic("plugins: Register plugin is nil")
	}

	if _, dup := plugins[name]; dup {
		panic("plugins: Register called twice for plugin " + name)
	}
	plugins[name] = plugin
	registerConfig(name, cfg)
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

func registerConfig(name string, cfg interface{}) {
	configsMu.Lock()
	defer configsMu.Unlock()
	if cfg == nil {
		panic("plugins: Register config is nil")
	}

	if _, dup := configs[name]; dup {
		panic("plugins: Register called twice for config " + name)
	}
	configs[name] = cfg
}

func DefaultConfigs() map[string]interface{} {
	configsMu.Lock()
	defer configsMu.Unlock()

	cpy := make(map[string]interface{}, len(configs))
	for k, v := range configs {
		cpy[k] = deepcopy.Copy(v)
	}
	return cpy
}
