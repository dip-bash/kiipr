// Package config resolves kiipr's runtime settings: XDG paths, limits and
// backend selection. Environment variables provide defaults, command-line
// flags (applied by the cli package) override them.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Defaults for the tunable limits.
const (
	DefaultLimit    = 500
	DefaultMaxBytes = 1 << 20  // 1 MiB: larger clipboard text is not recorded
	DefaultMaxMedia = 16 << 20 // 16 MiB: larger images are not recorded
	// DefaultMediaBudget caps the total size of stored images; the oldest
	// unpinned ones are dropped first.
	DefaultMediaBudget = 256 << 20
)

// Config is the effective configuration of one kiipr invocation.
type Config struct {
	DBPath   string // SQLite database
	StateDir string // pid file and log
	Limit    int    // max clipboard entries kept (pinned entries are exempt)
	MaxBytes int    // skip clipboard text larger than this
	MaxMedia int    // skip images larger than this
	Backend  string // "auto", "wayland", "x11" or "tty"
}

// legacyName is the name the project had before it was renamed to kiipr.
// It survives only in the old database location and environment variables.
const legacyName = "suchi"

// Default builds a Config from XDG directories and KIIPR_* environment
// variables. Settings and the history of a pre-rename installation are
// adopted automatically, so upgrading needs no manual step.
func Default() Config {
	return Config{
		DBPath:   defaultDBPath(),
		StateDir: filepath.Join(stateHome(), "kiipr"),
		Limit:    envInt("KIIPR_LIMIT", "SUCHI_LIMIT", DefaultLimit),
		MaxBytes: envInt("KIIPR_MAX_BYTES", "SUCHI_MAX_BYTES", DefaultMaxBytes),
		MaxMedia: envInt("KIIPR_MAX_MEDIA", "SUCHI_MAX_MEDIA", DefaultMaxMedia),
		Backend:  envStr("KIIPR_BACKEND", "SUCHI_BACKEND", "auto"),
	}
}

func defaultDBPath() string {
	if v := envStr("KIIPR_DB", "SUCHI_DB", ""); v != "" {
		return v
	}
	return adoptLegacyDB(filepath.Join(dataHome(), "kiipr", "kiipr.db"))
}

// adoptLegacyDB moves the database of a pre-rename installation into place
// the first time kiipr runs, so an upgrade keeps the clipboard history. If the
// move fails the old path is used as is, which is still better than starting
// with an empty history.
func adoptLegacyDB(path string) string {
	if fileExists(path) {
		return path
	}
	legacy := filepath.Join(dataHome(), legacyName, legacyName+".db")
	if !fileExists(legacy) {
		return path
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return legacy
	}
	// SQLite keeps its WAL and shared-memory files next to the database, so
	// move them together and lose nothing that was written recently.
	for _, suffix := range []string{"", "-wal", "-shm"} {
		from := legacy + suffix
		if fileExists(from) {
			_ = os.Rename(from, path+suffix)
		}
	}
	if !fileExists(path) {
		return legacy
	}
	return path
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// PIDPath is the daemon's lock/pid file.
func (c Config) PIDPath() string { return filepath.Join(c.StateDir, "kiipr.pid") }

// UIPIDPath is the lock of the open interface: only one kiipr window runs at a time.
func (c Config) UIPIDPath() string { return filepath.Join(c.StateDir, "kiipr-ui.pid") }

// LogPath is the daemon's log file.
func (c Config) LogPath() string { return filepath.Join(c.StateDir, "kiipr.log") }

// Validate reports the first invalid setting.
func (c Config) Validate() error {
	switch c.Backend {
	case "auto", "wayland", "x11", "tty":
	default:
		return fmt.Errorf("backend must be auto, wayland, x11 or tty, got %q", c.Backend)
	}
	if c.Limit < 1 {
		return fmt.Errorf("limit must be at least 1")
	}
	if c.MaxBytes < 1 {
		return fmt.Errorf("max-size must be at least 1")
	}
	if c.MaxMedia < 0 {
		return fmt.Errorf("max-media must not be negative")
	}
	return nil
}

func dataHome() string {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return v
	}
	return filepath.Join(home(), ".local", "share")
}

func stateHome() string {
	if v := os.Getenv("XDG_STATE_HOME"); v != "" {
		return v
	}
	return filepath.Join(home(), ".local", "state")
}

func home() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return "."
}

// envStr returns the value of key, or of legacyKey when key is unset, or def.
func envStr(key, legacyKey, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	if v := os.Getenv(legacyKey); v != "" {
		return v
	}
	return def
}

func envInt(key, legacyKey string, def int) int {
	for _, k := range []string{key, legacyKey} {
		if v := os.Getenv(k); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				return n
			}
		}
	}
	return def
}
