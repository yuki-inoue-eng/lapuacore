package configs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/fsnotify/fsnotify"
)

const warnInterval = 1 * time.Minute

// Watcher monitors config and secret files for changes using fsnotify.
type Watcher struct {
	watcher *fsnotify.Watcher

	configFilePath string
	secretFilePath string

	cancelCause context.CancelCauseFunc

	config *Config
	secret *Secret
}

// NewWatcher creates a new Watcher that reads and watches the given config and secret files.
// cancelCause is called when file watching breaks (e.g. failed to re-watch after rename),
// triggering a graceful shutdown of the application.
func NewWatcher(cancelCause context.CancelCauseFunc, configFilePath, secretFilePath string) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	if err = fsWatcher.Add(configFilePath); err != nil {
		return nil, fmt.Errorf("failed to watch config file: %w", err)
	}

	if err = fsWatcher.Add(secretFilePath); err != nil {
		return nil, fmt.Errorf("failed to watch secret file: %w", err)
	}

	rawConf, err := readRawConfig(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	rawSecret, err := readRawSecret(secretFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret file: %w", err)
	}

	w := &Watcher{
		watcher:        fsWatcher,
		cancelCause:    cancelCause,
		config:         newConfig(rawConf),
		secret:         newSecret(rawSecret),
		configFilePath: configFilePath,
		secretFilePath: secretFilePath,
	}

	return w, nil
}

// Start begins watching config and secret files for changes.
// It blocks until the context is cancelled.
func (w *Watcher) Start(ctx context.Context) {
	defer w.watcher.Close()
	ticker := time.NewTicker(warnInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-w.watcher.Events:
			w.handleEvent(event)
		case <-ticker.C:
			w.config.Params.logFailedKeys()
		}
	}
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	switch event.Name {
	case w.configFilePath:
		w.handleConfigEvent(event)
	case w.secretFilePath:
		w.handleSecretEvent(event)
	}
}

func (w *Watcher) handleConfigEvent(event fsnotify.Event) {
	// vim replaces the file on save, which triggers a Rename event.
	// Re-add the file path to continue watching.
	if event.Has(fsnotify.Rename) {
		if err := w.resetFilePath(w.configFilePath); err != nil {
			slog.Error("failed to re-watch config file, shutting down", "error", err)
			w.cancelCause(fmt.Errorf("failed to re-watch config file: %w", err))
			return
		}
	}
	if event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) {
		if err := w.updateConfig(); err != nil {
			slog.Warn("failed to update config, keeping old config", "error", err)
			return
		}
		slog.Info("config file updated")
	}
}

func (w *Watcher) handleSecretEvent(event fsnotify.Event) {
	// vim replaces the file on save, which triggers a Rename event.
	if event.Has(fsnotify.Rename) {
		if err := w.resetFilePath(w.secretFilePath); err != nil {
			slog.Error("failed to re-watch secret file, shutting down", "error", err)
			w.cancelCause(fmt.Errorf("failed to re-watch secret file: %w", err))
			return
		}
	}
	if event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) {
		if err := w.updateSecretFromFile(); err != nil {
			slog.Warn("failed to update secret, keeping old secret", "error", err)
			return
		}
		slog.Info("secret file updated")
	}
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
