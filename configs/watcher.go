package configs

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors config and secret files for changes using fsnotify.
type Watcher struct {
	watcher *fsnotify.Watcher

	configFilePath string
	secretFilePath string

	config *Config
	secret *Secret
}

// NewWatcher creates a new Watcher that reads and watches the given config and secret files.
func NewWatcher(configFilePath, secretFilePath string) *Watcher {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

	if err = fsWatcher.Add(configFilePath); err != nil {
		panic(err)
	}

	if err = fsWatcher.Add(secretFilePath); err != nil {
		panic(err)
	}

	rawConf, err := readRawConfig(configFilePath)
	if err != nil {
		panic(err)
	}

	rawSecret, err := readRawSecret(secretFilePath)
	if err != nil {
		panic(err)
	}

	w := &Watcher{
		watcher:        fsWatcher,
		config:         newConfig(rawConf),
		secret:         newSecret(rawSecret),
		configFilePath: configFilePath,
		secretFilePath: secretFilePath,
	}

	return w
}

// Start begins watching config and secret files for changes.
// It blocks until the context is cancelled.
func (w *Watcher) Start(ctx context.Context) {
	defer w.watcher.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-w.watcher.Events:
			w.handleEvent(event)
		}
	}
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	switch event.Name {
	case w.configFilePath:
		if err := w.configFileEventHandler(event); err != nil {
			slog.Error(err.Error())
			return
		}
		slog.Info("config file updated")
	case w.secretFilePath:
		if err := w.secretFileEventHandler(event); err != nil {
			slog.Error(err.Error())
			return
		}
		slog.Info("secret file updated")
	}
}

func (w *Watcher) configFileEventHandler(event fsnotify.Event) error {
	if event.Has(fsnotify.Write) {
		if err := w.updateConfig(); err != nil {
			return err
		}
	}
	// vim replaces the file on save, which triggers a Rename event.
	// Re-add the file path to continue watching.
	if event.Has(fsnotify.Rename) {
		if err := w.resetFilePath(w.configFilePath); err != nil {
			return err
		}
		if err := w.updateConfig(); err != nil {
			return err
		}
	}
	return nil
}

func (w *Watcher) secretFileEventHandler(event fsnotify.Event) error {
	if event.Has(fsnotify.Write) {
		if err := w.updateSecretFromFile(); err != nil {
			return err
		}
	}
	// vim replaces the file on save, which triggers a Rename event.
	if event.Has(fsnotify.Rename) {
		if err := w.resetFilePath(w.secretFilePath); err != nil {
			return err
		}
		if err := w.updateSecretFromFile(); err != nil {
			return err
		}
	}
	return nil
}

func (w *Watcher) resetFilePath(filePath string) error {
	if err := w.watcher.Add(filePath); err != nil {
		return fmt.Errorf("failed to re-add file path: %v", err)
	}
	return nil
}

func (w *Watcher) updateConfig() error {
	rawConf, err := readRawConfig(w.configFilePath)
	if err != nil {
		return err
	}
	w.config.update(rawConf)
	return nil
}

func (w *Watcher) updateSecretFromFile() error {
	rawSecret, err := readRawSecret(w.secretFilePath)
	if err != nil {
		return err
	}
	w.secret.update(rawSecret)
	return nil
}

// GetConfig returns the current Config.
func (w *Watcher) GetConfig() *Config {
	return w.config
}

// GetSecret returns the current Secret.
func (w *Watcher) GetSecret() *Secret {
	return w.secret
}
