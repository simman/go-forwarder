package config

import (
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

// Watcher monitors configuration file changes
type Watcher struct {
	configPath string
	onChange   func(*Config) error
	watcher    *fsnotify.Watcher
	mu         sync.Mutex
	stopped    bool
}

// NewWatcher creates a new configuration file watcher
func NewWatcher(configPath string, onChange func(*Config) error) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	w := &Watcher{
		configPath: configPath,
		onChange:   onChange,
		watcher:    watcher,
	}

	return w, nil
}

// Start begins watching the configuration file
func (w *Watcher) Start() error {
	if err := w.watcher.Add(w.configPath); err != nil {
		return fmt.Errorf("failed to watch config file: %w", err)
	}

	go w.watch()

	log.Info().Str("path", w.configPath).Msg("config watcher started")
	return nil
}

// Stop stops the watcher
func (w *Watcher) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return nil
	}

	w.stopped = true
	if err := w.watcher.Close(); err != nil {
		return fmt.Errorf("failed to close watcher: %w", err)
	}

	log.Info().Msg("config watcher stopped")
	return nil
}

// watch monitors file system events
func (w *Watcher) watch() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Handle file write or create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				log.Info().Str("file", event.Name).Str("op", event.Op.String()).Msg("config file changed, reloading")
				w.reload()
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Error().Err(err).Msg("watcher error")
		}
	}
}

// reload loads and applies the new configuration
func (w *Watcher) reload() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return
	}

	// Load new config
	cfg, err := LoadConfig(w.configPath)
	if err != nil {
		log.Error().Err(err).Msg("failed to reload config, keeping old config")
		return
	}

	// Apply new config
	if err := w.onChange(cfg); err != nil {
		log.Error().Err(err).Msg("failed to apply new config, keeping old config")
		return
	}

	log.Info().Msg("config reloaded successfully")
}
