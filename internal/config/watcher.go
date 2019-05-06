package config

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"sync"

	"code.vegaprotocol.io/vega/internal/logging"
	"github.com/fsnotify/fsnotify"
	"github.com/zannen/toml"
)

const (
	configFileName = "config.toml"
	namedLogger    = "cfgwatcher"
)

type Watcher struct {
	log  *logging.Logger
	cfg  Config
	path string

	cfgUpdateListeners []func(Config)
	mu                 sync.Mutex
}

func NewFromFile(ctx context.Context, log *logging.Logger, defaultStoreDirPath string, path string) (*Watcher, error) {
	watcherlog := log.Named(namedLogger)
	// set this logger to debug level as we want to be notified for any configuration changes at any time
	watcherlog.SetLevel(logging.DebugLevel)
	w := &Watcher{
		log:                watcherlog,
		cfg:                NewDefaultConfig(defaultStoreDirPath),
		path:               filepath.Join(path, configFileName),
		cfgUpdateListeners: []func(Config){},
	}

	err := w.load()
	if err != nil {
		return nil, err
	}

	err = w.watch(ctx)
	if err != nil {
		return nil, err
	}

	return w, nil
}

func (w *Watcher) Get() Config {
	w.mu.Lock()
	conf := w.cfg
	w.mu.Unlock()
	return conf
}

func (w *Watcher) OnConfigUpdate(f func(Config)) {
	w.mu.Lock()
	w.cfgUpdateListeners = append(w.cfgUpdateListeners, f)
	w.mu.Unlock()
}

func (w *Watcher) notifyCfgUpdate() {
	w.mu.Lock()
	for _, f := range w.cfgUpdateListeners {
		f(w.cfg)
	}
	w.mu.Unlock()
}

func (w *Watcher) load() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	buf, err := ioutil.ReadFile(w.path)
	if err != nil {
		return err
	}
	if _, err := toml.Decode(string(buf), &w.cfg); err != nil {
		return err
	}
	return nil
}

func (w *Watcher) watch(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	err = watcher.Add(w.path)
	if err != nil {
		return err
	}

	go func(log *logging.Logger) {
		defer watcher.Close()
		for {
			select {
			case event, _ := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Info("configuration updated", logging.String("event", event.Name))
					err := w.load()
					if err != nil {
						log.Error("unable to load configuration", logging.Error(err))
						continue
					}
					w.notifyCfgUpdate()
				}
			case err, _ := <-watcher.Errors:
				log.Error("config watcher received error event", logging.Error(err))
			case <-ctx.Done():
				log.Error("ctx done", logging.Error(err))
				return
			}
		}
	}(w.log)
	return nil
}
