package config

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/internal/logging"
	"github.com/fsnotify/fsnotify"
	"github.com/zannen/toml"
)

const (
	configFileName = "config.toml"
	namedLogger    = "cfgwatcher"
)

// Watcher is looking for updates in the configurations files
type Watcher struct {
	log  *logging.Logger
	cfg  Config
	path string

	cfgUpdateListeners []func(Config)
	mu                 sync.Mutex
}

// NewFromFile instanciate a new watcher from the vega config files
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

// Get return the last update of the configuration
func (w *Watcher) Get() Config {
	w.mu.Lock()
	conf := w.cfg
	w.mu.Unlock()
	return conf
}

// OnConfigUpdate register a function to be called when the configuration is getting updated
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

	w.log.Info("config watcher started successfully",
		logging.String("config", w.path))

	go func(log *logging.Logger) {
		defer watcher.Close()
		for {
			select {
			case event, _ := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Rename == fsnotify.Rename {
					if event.Op&fsnotify.Rename == fsnotify.Rename {
						// add a small sleep here in order to handle vi
						// vi do not send a write event / edit the file in place,
						// it always create a temporary file, then delete the original one,
						// and then rename the temp file with the name of the original file.
						// if we try to update the conf as soon as we get the event, the file is not
						// always created and we get a no such file or directory error
						time.Sleep(time.Duration(50 * time.Millisecond))
					}
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
				log.Error("config watcher ctx done", logging.Error(err))
				return
			}
		}
	}(w.log)
	return nil
}
