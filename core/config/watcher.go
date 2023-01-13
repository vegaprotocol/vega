// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package config

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	"github.com/fsnotify/fsnotify"
)

const (
	namedLogger = "cfgwatcher"
)

// Watcher is looking for updates in the configurations files.
type Watcher struct {
	log            *logging.Logger
	cfg            Config
	configFilePath string

	// to be used as an atomic
	hasChanged         atomic.Bool
	cfgUpdateListeners []func(Config)
	cfgHandlers        []func(*Config) error

	// listeners with IDs
	cfgUpdateListenersWithID map[int]func(Config)
	currentID                int
	mu                       sync.Mutex
}

type Option func(w *Watcher)

func Use(use func(*Config) error) Option {
	fn := func(w *Watcher) {
		w.Use(use)
	}

	return fn
}

// NewWatcher instantiate a new watcher from the vega config files.
func NewWatcher(ctx context.Context, log *logging.Logger, vegaPaths paths.Paths, opts ...Option) (*Watcher, error) {
	watcherLog := log.Named(namedLogger)
	// set this logger to debug level as we want to be notified for any configuration changes at any time
	watcherLog.SetLevel(logging.DebugLevel)

	configFilePath, err := vegaPaths.CreateConfigPathFor(paths.NodeDefaultConfigFile)
	if err != nil {
		return nil, fmt.Errorf("couldn't get path for %s: %w", paths.NodeDefaultConfigFile, err)
	}

	w := &Watcher{
		log:                      watcherLog,
		cfg:                      NewDefaultConfig(),
		configFilePath:           configFilePath,
		cfgUpdateListeners:       []func(Config){},
		cfgUpdateListenersWithID: map[int]func(Config){},
	}

	for _, opt := range opts {
		opt(w)
	}

	err = w.load()
	if err != nil {
		return nil, err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	err = watcher.Add(w.configFilePath)
	if err != nil {
		return nil, err
	}

	go w.watch(ctx, watcher)

	return w, nil
}

func (w *Watcher) OnTimeUpdate(_ context.Context, _ time.Time) {
	if !w.hasChanged.Load() {
		// no changes we can return straight away
		return
	}
	// get the config and updates listeners
	cfg := w.Get()

	for _, f := range w.cfgUpdateListeners {
		f(cfg)
	}

	ids := []int{}
	for k := range w.cfgUpdateListenersWithID {
		ids = append(ids, k)
	}
	sort.Ints(ids)

	for id := range ids {
		w.cfgUpdateListenersWithID[id](cfg)
	}

	// reset the atomic
	w.hasChanged.Store(false)
}

// Get return the last update of the configuration.
func (w *Watcher) Get() Config {
	w.mu.Lock()
	conf := w.cfg
	w.mu.Unlock()
	return conf
}

// OnConfigUpdate register a function to be called when the configuration is getting updated.
func (w *Watcher) OnConfigUpdate(fns ...func(Config)) {
	w.mu.Lock()
	w.cfgUpdateListeners = append(w.cfgUpdateListeners, fns...)
	w.mu.Unlock()
}

// OnConfigUpdate register a function to be called when the configuration is getting updated.
func (w *Watcher) OnConfigUpdateWithID(fns ...func(Config)) []int {
	w.mu.Lock()
	// w.cfgUpdateListeners = append(w.cfgUpdateListeners, fns...)
	ids := []int{}
	for _, f := range fns {
		id := w.currentID
		ids = append(ids, id)
		w.cfgUpdateListenersWithID[id] = f
		w.currentID++
	}
	w.mu.Unlock()
	return ids
}

func (w *Watcher) Unregister(ids []int) {
	for _, id := range ids {
		delete(w.cfgUpdateListenersWithID, id)
	}
}

// Use registers a function that modify the config when the configuration is updated.
func (w *Watcher) Use(fns ...func(*Config) error) {
	w.mu.Lock()
	w.cfgHandlers = append(w.cfgHandlers, fns...)
	w.mu.Unlock()
}

func (w *Watcher) load() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := paths.ReadStructuredFile(w.configFilePath, &w.cfg); err != nil {
		return fmt.Errorf("couldn't read configuration file at %s: %w", w.configFilePath, err)
	}

	for _, f := range w.cfgHandlers {
		if err := f(&w.cfg); err != nil {
			return err
		}
	}

	return nil
}

func (w *Watcher) watch(ctx context.Context, watcher *fsnotify.Watcher) {
	defer watcher.Close()
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Rename == fsnotify.Rename {
				if event.Op&fsnotify.Rename == fsnotify.Rename {
					// add a small sleep here in order to handle vi
					// vi do not send a write event / edit the file in place,
					// it always create a temporary file, then delete the original one,
					// and then rename the temp file with the name of the original file.
					// if we try to update the conf as soon as we get the event, the file is not
					// always created and we get a no such file or directory error
					time.Sleep(50 * time.Millisecond)
				}
				w.log.Info("configuration updated", logging.String("event", event.Name))
				err := w.load()
				if err != nil {
					w.log.Error("unable to load configuration", logging.Error(err))
					continue
				}
				w.hasChanged.Store(true)
			}
		case err := <-watcher.Errors:
			w.log.Error("config watcher received error event", logging.Error(err))
		case <-ctx.Done():
			w.log.Error("config watcher ctx done")
			return
		}
	}
}
