package dml

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const watchTestTimeout = 2 * time.Second

var watchMtimeTick atomic.Int64

func newWatchChannels() (chan *Config, chan error, WatchOptions) {
	reloaded := make(chan *Config, 64)
	errs := make(chan error, 64)
	opts := WatchOptions{
		Interval: 10 * time.Millisecond,
		OnReload: func(c *Config) {
			select {
			case reloaded <- c:
			default:
			}
		},
		OnError: func(err error) {
			select {
			case errs <- err:
			default:
			}
		},
	}
	return reloaded, errs, opts
}

func startWatch(t *testing.T, path string, opts WatchOptions) *Watcher {
	t.Helper()
	w, err := Watch(path, opts)
	if err != nil {
		t.Fatalf("Watch: %v", err)
	}
	t.Cleanup(w.Stop)
	return w
}

func rewriteDML(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	mtime := time.Unix(1_700_000_000+watchMtimeTick.Add(1), 0)
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
}

func awaitReload(t *testing.T, ch <-chan *Config, accept func(*Config) bool) *Config {
	t.Helper()
	deadline := time.After(watchTestTimeout)
	for {
		select {
		case cfg := <-ch:
			if accept(cfg) {
				return cfg
			}
		case <-deadline:
			t.Fatalf("timed out waiting for reload")
			return nil
		}
	}
}

func awaitError(t *testing.T, ch <-chan error) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(watchTestTimeout):
		t.Fatalf("timed out waiting for error")
		return nil
	}
}

func appIs(want string) func(*Config) bool {
	return func(c *Config) bool { return c.GetString("app") == want }
}

func expectQuiet(t *testing.T, reloaded <-chan *Config, errs <-chan error) {
	t.Helper()
	select {
	case <-reloaded:
		t.Fatalf("unexpected reload")
	case err := <-errs:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestWatch_InitialLoad(t *testing.T) {
	path := writeTempDML(t, "string app = \"v1\";\nint port = 8080;\n")

	w := startWatch(t, path, WatchOptions{Interval: 10 * time.Millisecond})

	if got := w.GetString("app"); got != "v1" {
		t.Fatalf("expected app=v1, got %q", got)
	}
	if got := w.GetInt("port"); got != 8080 {
		t.Fatalf("expected port=8080, got %d", got)
	}
	if !w.Has("app") {
		t.Fatalf("expected Has(app) to be true")
	}
	if len(w.Keys()) != 2 {
		t.Fatalf("expected 2 keys, got %v", w.Keys())
	}
	if v, ok := w.Get("app"); !ok || v != "v1" {
		t.Fatalf("expected Get(app)=v1, got %v %v", v, ok)
	}
}

func TestWatch_ReloadsOnChange(t *testing.T) {
	path := writeTempDML(t, "string app = \"v1\";\n")
	reloaded, _, opts := newWatchChannels()
	w := startWatch(t, path, opts)
	before := w.Config()

	rewriteDML(t, path, "string app = \"v2\";\nbool extra = true;\n")

	cfg := awaitReload(t, reloaded, appIs("v2"))
	if !cfg.GetBool("extra") {
		t.Fatalf("expected extra=true in reloaded config")
	}
	if got := w.GetString("app"); got != "v2" {
		t.Fatalf("expected watcher app=v2, got %q", got)
	}
	if got := before.GetString("app"); got != "v1" {
		t.Fatalf("expected earlier snapshot to stay v1, got %q", got)
	}
}

func TestWatch_AtomicReplace(t *testing.T) {
	path := writeTempDML(t, "string app = \"v1\";\n")
	reloaded, _, opts := newWatchChannels()
	w := startWatch(t, path, opts)

	replacement := filepath.Join(filepath.Dir(path), "replacement.dml")
	if err := os.WriteFile(replacement, []byte("string app = \"v2\";\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Rename(replacement, path); err != nil {
		t.Fatalf("Rename: %v", err)
	}

	awaitReload(t, reloaded, appIs("v2"))
	if got := w.GetString("app"); got != "v2" {
		t.Fatalf("expected app=v2, got %q", got)
	}
}

func TestWatch_KeepsLastGoodConfigOnParseError(t *testing.T) {
	path := writeTempDML(t, "string app = \"v1\";\n")
	reloaded, errs, opts := newWatchChannels()
	w := startWatch(t, path, opts)

	rewriteDML(t, path, "int port = abc;\n")

	err := awaitError(t, errs)
	var dmlErr *DMLError
	if !errors.As(err, &dmlErr) {
		t.Fatalf("expected a DMLError, got %T: %v", err, err)
	}
	if got := w.GetString("app"); got != "v1" {
		t.Fatalf("expected last good config to be kept, got app=%q", got)
	}
	expectQuiet(t, reloaded, errs)

	rewriteDML(t, path, "string app = \"v3\";\n")

	awaitReload(t, reloaded, appIs("v3"))
	if got := w.GetString("app"); got != "v3" {
		t.Fatalf("expected recovery to v3, got %q", got)
	}
}

func TestWatch_ErrorDoesNotLeakFileContents(t *testing.T) {
	const marker = "TOPSECRETMARKER"

	t.Run("initial load", func(t *testing.T) {
		path := writeTempDML(t, "int port = "+marker+";\n")

		_, err := Watch(path, WatchOptions{})
		if err == nil {
			t.Fatalf("expected an error")
		}
		if strings.Contains(err.Error(), marker) {
			t.Fatalf("error leaks file contents: %v", err)
		}
	})

	t.Run("reload", func(t *testing.T) {
		path := writeTempDML(t, "string app = \"v1\";\n")
		_, errs, opts := newWatchChannels()
		startWatch(t, path, opts)

		rewriteDML(t, path, "string app = \"v1\";\nint port = "+marker+";\n")

		err := awaitError(t, errs)
		if strings.Contains(err.Error(), marker) {
			t.Fatalf("error leaks file contents: %v", err)
		}
		var dmlErr *DMLError
		if !errors.As(err, &dmlErr) {
			t.Fatalf("expected a DMLError, got %T", err)
		}
		if dmlErr.Line != 2 {
			t.Fatalf("expected line 2, got %d", dmlErr.Line)
		}
		if dmlErr.Context != "" {
			t.Fatalf("expected empty context, got %q", dmlErr.Context)
		}
	})
}

func TestWatch_RejectsInvalidPath(t *testing.T) {
	for _, path := range []string{"", "../config.dml", "dir/../config.dml", "bad\x00.dml"} {
		w, err := Watch(path, WatchOptions{})
		if err == nil {
			w.Stop()
			t.Fatalf("expected an error for %q", path)
		}
		if !errors.Is(err, ErrInvalidPath) {
			t.Fatalf("expected ErrInvalidPath for %q, got %v", path, err)
		}
	}
}

func TestWatch_MissingFileAtStart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.dml")

	w, err := Watch(path, WatchOptions{})
	if err == nil {
		w.Stop()
		t.Fatalf("expected an error")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected ErrNotExist, got %v", err)
	}
}

func TestWatch_RejectsOversizedFile(t *testing.T) {
	t.Run("initial load", func(t *testing.T) {
		path := writeTempDML(t, "string app = \"a-value-longer-than-the-limit\";\n")

		w, err := Watch(path, WatchOptions{MaxFileSize: 16})
		if err == nil {
			w.Stop()
			t.Fatalf("expected an error")
		}
		if !errors.Is(err, ErrWatchFileTooLarge) {
			t.Fatalf("expected ErrWatchFileTooLarge, got %v", err)
		}
	})

	t.Run("reload", func(t *testing.T) {
		path := writeTempDML(t, "string a = \"1\";\n")
		_, errs, opts := newWatchChannels()
		opts.MaxFileSize = 64
		w := startWatch(t, path, opts)

		rewriteDML(t, path, "string a = \""+strings.Repeat("x", 200)+"\";\n")

		err := awaitError(t, errs)
		if !errors.Is(err, ErrWatchFileTooLarge) {
			t.Fatalf("expected ErrWatchFileTooLarge, got %v", err)
		}
		if got := w.GetString("a"); got != "1" {
			t.Fatalf("expected last good config to be kept, got %q", got)
		}
	})
}

func TestWatch_FileMissingKeepsConfig(t *testing.T) {
	path := writeTempDML(t, "string app = \"v1\";\n")
	reloaded, errs, opts := newWatchChannels()
	w := startWatch(t, path, opts)

	if err := os.Remove(path); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	err := awaitError(t, errs)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected ErrNotExist, got %v", err)
	}
	if got := w.GetString("app"); got != "v1" {
		t.Fatalf("expected last good config to be kept, got %q", got)
	}
	expectQuiet(t, reloaded, errs)

	rewriteDML(t, path, "string app = \"v2\";\n")

	awaitReload(t, reloaded, appIs("v2"))
}

func TestWatch_StopIsIdempotentAndSilencesCallbacks(t *testing.T) {
	path := writeTempDML(t, "string app = \"v1\";\n")
	reloaded, errs, opts := newWatchChannels()
	w := startWatch(t, path, opts)

	w.Stop()
	w.Stop()

	rewriteDML(t, path, "string app = \"v2\";\n")
	expectQuiet(t, reloaded, errs)

	if got := w.GetString("app"); got != "v1" {
		t.Fatalf("expected config to stay readable and unchanged, got %q", got)
	}
}

func TestWatch_CallbackCanReadWatcher(t *testing.T) {
	path := writeTempDML(t, "string app = \"v1\";\n")
	seen := make(chan string, 8)
	var holder atomic.Pointer[Watcher]

	opts := WatchOptions{
		Interval: 10 * time.Millisecond,
		OnReload: func(*Config) {
			if w := holder.Load(); w != nil {
				seen <- w.GetString("app")
			}
		},
	}
	w := startWatch(t, path, opts)
	holder.Store(w)

	rewriteDML(t, path, "string app = \"v2\";\n")

	deadline := time.After(watchTestTimeout)
	for {
		select {
		case got := <-seen:
			if got == "v2" {
				return
			}
		case <-deadline:
			t.Fatalf("callback never observed the new value")
		}
	}
}

func TestWatch_ConcurrentReaders(t *testing.T) {
	path := writeTempDML(t, "int n = 0;\nstring app = \"v\";\n")
	w := startWatch(t, path, WatchOptions{Interval: 5 * time.Millisecond})

	const readers = 20
	const versions = 30

	stopReaders := make(chan struct{})
	var wg sync.WaitGroup
	var outOfRange atomic.Int64

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopReaders:
					return
				default:
				}
				n := w.GetInt("n")
				if n < 0 || n > versions {
					outOfRange.Add(1)
				}
				w.GetString("app")
				w.Has("n")
				w.Keys()
				w.Config().GetMap("missing")
			}
		}()
	}

	for i := 1; i <= versions; i++ {
		rewriteDML(t, path, fmt.Sprintf("int n = %d;\nstring app = \"v%d\";\n", i, i))
		time.Sleep(8 * time.Millisecond)
	}

	deadline := time.Now().Add(watchTestTimeout)
	for w.GetInt("n") != versions && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	close(stopReaders)
	wg.Wait()

	if got := w.GetInt("n"); got != versions {
		t.Fatalf("expected final n=%d, got %d", versions, got)
	}
	if outOfRange.Load() != 0 {
		t.Fatalf("readers observed %d out-of-range values", outOfRange.Load())
	}
}

type watchedConfig struct {
	App  string `dml:"app"`
	Port int    `dml:"port" validate:"min=1"`
}

func validateWatchedConfig(c *Config) error {
	_, err := Decode[watchedConfig](c)
	return err
}

func TestWatch_ValidateRejectsInitialLoad(t *testing.T) {
	path := writeTempDML(t, "string app = \"v1\";\nint port = 0;\n")

	w, err := Watch(path, WatchOptions{Interval: 10 * time.Millisecond, Validate: validateWatchedConfig})
	if err == nil {
		w.Stop()
		t.Fatalf("expected a validation error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestWatch_ValidateAcceptsInitialLoad(t *testing.T) {
	path := writeTempDML(t, "string app = \"v1\";\nint port = 8080;\n")

	w := startWatch(t, path, WatchOptions{Interval: 10 * time.Millisecond, Validate: validateWatchedConfig})

	if got := w.GetString("app"); got != "v1" {
		t.Fatalf("expected app=v1, got %q", got)
	}
}

func TestWatch_KeepsLastGoodConfigOnValidationError(t *testing.T) {
	path := writeTempDML(t, "string app = \"v1\";\nint port = 8080;\n")
	reloaded, errs, opts := newWatchChannels()
	opts.Validate = validateWatchedConfig
	w := startWatch(t, path, opts)

	rewriteDML(t, path, "string app = \"v2\";\nint port = 0;\n")

	err := awaitError(t, errs)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
	if got := w.GetString("app"); got != "v1" {
		t.Fatalf("expected last good config to be kept, got app=%q", got)
	}
	expectQuiet(t, reloaded, errs)

	rewriteDML(t, path, "string app = \"v3\";\nint port = 9090;\n")

	awaitReload(t, reloaded, appIs("v3"))
	if got := w.GetString("app"); got != "v3" {
		t.Fatalf("expected recovery to v3, got %q", got)
	}
}
