package dml

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

const (
	defaultWatchInterval    = time.Second
	defaultWatchMaxFileSize = 10 << 20
	watchRedactedMessage    = "invalid content, see the reported position in the file"
)

var ErrWatchFileTooLarge = errors.New("file exceeds the maximum watch size")

type WatchOptions struct {
	Interval    time.Duration
	MaxFileSize int64
	OnReload    func(*Config)
	OnError     func(error)
	Validate    func(*Config) error
}

type Watcher struct {
	mu       sync.RWMutex
	cfg      *Config
	path     string
	opts     WatchOptions
	last     os.FileInfo
	pending  os.FileInfo
	missing  bool
	stop     chan struct{}
	done     chan struct{}
	stopOnce sync.Once
}

func Watch(filepath string, opts WatchOptions) (*Watcher, error) {
	if err := validatePath(filepath); err != nil {
		return nil, err
	}
	if opts.Interval <= 0 {
		opts.Interval = defaultWatchInterval
	}
	if opts.MaxFileSize <= 0 {
		opts.MaxFileSize = defaultWatchMaxFileSize
	}

	info, err := os.Stat(filepath)
	if err != nil {
		return nil, fmt.Errorf("dml watch: %w", err)
	}

	cfg, err := loadWatched(filepath, opts.MaxFileSize)
	if err != nil {
		return nil, redactWatchError(err)
	}
	if err := validateWatched(opts, cfg); err != nil {
		return nil, err
	}

	w := &Watcher{
		cfg:  cfg,
		path: filepath,
		opts: opts,
		last: info,
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
	go w.run()

	return w, nil
}

func (w *Watcher) Config() *Config {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cfg
}

func (w *Watcher) Get(key string) (any, bool) {
	return w.Config().Get(key)
}

func (w *Watcher) GetString(key string) string {
	return w.Config().GetString(key)
}

func (w *Watcher) GetInt(key string) int {
	return w.Config().GetInt(key)
}

func (w *Watcher) GetFloat(key string) float64 {
	return w.Config().GetFloat(key)
}

func (w *Watcher) GetBool(key string) bool {
	return w.Config().GetBool(key)
}

func (w *Watcher) GetList(key string) []any {
	return w.Config().GetList(key)
}

func (w *Watcher) GetMap(key string) map[string]any {
	return w.Config().GetMap(key)
}

func (w *Watcher) Has(key string) bool {
	return w.Config().Has(key)
}

func (w *Watcher) Keys() []string {
	return w.Config().Keys()
}

func (w *Watcher) Stop() {
	w.stopOnce.Do(func() { close(w.stop) })
	<-w.done
}

func (w *Watcher) run() {
	defer close(w.done)

	ticker := time.NewTicker(w.opts.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			w.poll()
		}
	}
}

func (w *Watcher) poll() {
	info, err := os.Stat(w.path)
	if err != nil {
		w.pending = nil
		if !w.missing {
			w.missing = true
			w.reportError(fmt.Errorf("dml watch: %w", err))
		}
		return
	}
	w.missing = false

	if !fileChanged(w.last, info) {
		w.pending = nil
		return
	}
	if w.pending == nil || fileChanged(w.pending, info) {
		w.pending = info
		return
	}
	w.pending = nil
	w.last = info

	cfg, err := loadWatched(w.path, w.opts.MaxFileSize)
	if err != nil {
		w.reportError(redactWatchError(err))
		return
	}
	if err := validateWatched(w.opts, cfg); err != nil {
		w.reportError(err)
		return
	}

	w.mu.Lock()
	w.cfg = cfg
	w.mu.Unlock()

	if w.opts.OnReload != nil {
		w.opts.OnReload(cfg)
	}
}

func (w *Watcher) reportError(err error) {
	if w.opts.OnError != nil {
		w.opts.OnError(err)
	}
}

func fileChanged(previous, current os.FileInfo) bool {
	return !os.SameFile(previous, current) ||
		!previous.ModTime().Equal(current.ModTime()) ||
		previous.Size() != current.Size()
}

func loadWatched(filepath string, maxSize int64) (*Config, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("dml watch: %w", err)
	}
	defer f.Close()

	content, err := io.ReadAll(io.LimitReader(f, maxSize+1))
	if err != nil {
		return nil, fmt.Errorf("dml watch: %w", err)
	}
	if int64(len(content)) > maxSize {
		return nil, fmt.Errorf("dml watch: %w", ErrWatchFileTooLarge)
	}

	cfg := New()
	if err := cfg.Parse(string(content)); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validateWatched(opts WatchOptions, cfg *Config) error {
	if opts.Validate == nil {
		return nil
	}
	if err := opts.Validate(cfg); err != nil {
		return fmt.Errorf("dml watch: %w", err)
	}
	return nil
}

func redactWatchError(err error) error {
	var dmlErr *DMLError
	if errors.As(err, &dmlErr) {
		return &DMLError{
			Line:    dmlErr.Line,
			Column:  dmlErr.Column,
			Message: watchRedactedMessage,
			Type:    dmlErr.Type,
		}
	}
	return err
}
