// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package config

import (
	"context"
	"fmt"
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
	mu                 sync.Mutex
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

	configFilePath, err := vegaPaths.CreateConfigPathFor(paths.DataNodeDefaultConfigFile)
	if err != nil {
		return nil, fmt.Errorf("couldn't get path for %s: %w", paths.NodeDefaultConfigFile, err)
	}

	w := &Watcher{
		log:                watcherLog,
		cfg:                NewDefaultConfig(),
		configFilePath:     configFilePath,
		cfgUpdateListeners: []func(Config){},
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

	w.log.Info("config watcher started successfully",
		logging.String("config", w.configFilePath))

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
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) {
				if event.Has(fsnotify.Rename) {
					// add a small sleep here in order to handle vi as
					// vi does not send a write event / edit the file in place,
					// it always creates a temporary file, then deletes the original one,
					// and then renames the temp file with the name of the original file.
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
