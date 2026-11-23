package config

import (
	"os"
	"path/filepath"
	"testing"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"KIIPR_DB", "SUCHI_DB",
		"KIIPR_LIMIT", "SUCHI_LIMIT",
		"KIIPR_MAX_BYTES", "SUCHI_MAX_BYTES",
		"KIIPR_MAX_MEDIA", "SUCHI_MAX_MEDIA",
		"KIIPR_BACKEND", "SUCHI_BACKEND",
	} {
		t.Setenv(k, "")
	}
}

func TestDefaultsFollowXDG(t *testing.T) {
	clearEnv(t)
	t.Setenv("XDG_DATA_HOME", "/data")
	t.Setenv("XDG_STATE_HOME", "/state")
	c := Default()
	if c.DBPath != filepath.Join("/data", "kiipr", "kiipr.db") ||
		c.PIDPath() != filepath.Join("/state", "kiipr", "kiipr.pid") ||
		c.LogPath() != filepath.Join("/state", "kiipr", "kiipr.log") {
		t.Fatalf("got %+v", c)
	}
	if c.Limit != DefaultLimit || c.MaxBytes != DefaultMaxBytes || c.Backend != "auto" {
		t.Fatalf("got %+v", c)
	}
}

func TestEnvOverrides(t *testing.T) {
	clearEnv(t)
	t.Setenv("KIIPR_DB", "/tmp/x.db")
	t.Setenv("KIIPR_LIMIT", "42")
	t.Setenv("KIIPR_MAX_BYTES", "not-a-number") // ignored, default kept
	c := Default()
	if c.DBPath != "/tmp/x.db" || c.Limit != 42 || c.MaxBytes != DefaultMaxBytes {
		t.Fatalf("got %+v", c)
	}
}

func TestLegacyEnvFallback(t *testing.T) {
	clearEnv(t)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Setenv("SUCHI_DB", "/tmp/old.db")
	t.Setenv("SUCHI_LIMIT", "7")
	t.Setenv("SUCHI_BACKEND", "x11")
	c := Default()
	if c.DBPath != "/tmp/old.db" || c.Limit != 7 || c.Backend != "x11" {
		t.Fatalf("got %+v", c)
	}

	// The new variables always win.
	t.Setenv("KIIPR_DB", "/tmp/new.db")
	t.Setenv("KIIPR_LIMIT", "9")
	t.Setenv("KIIPR_BACKEND", "wayland")
	c = Default()
	if c.DBPath != "/tmp/new.db" || c.Limit != 9 || c.Backend != "wayland" {
		t.Fatalf("got %+v", c)
	}
}

func TestAdoptsLegacyDatabase(t *testing.T) {
	clearEnv(t)
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	oldDir := filepath.Join(data, "suchi")
	if err := os.MkdirAll(oldDir, 0o700); err != nil {
		t.Fatal(err)
	}
	oldDB := filepath.Join(oldDir, "suchi.db")
	for name, body := range map[string]string{oldDB: "history", oldDB + "-wal": "wal"} {
		if err := os.WriteFile(name, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	c := Default()
	want := filepath.Join(data, "kiipr", "kiipr.db")
	if c.DBPath != want {
		t.Fatalf("DBPath = %q, want %q", c.DBPath, want)
	}
	for name, body := range map[string]string{want: "history", want + "-wal": "wal"} {
		got, err := os.ReadFile(name)
		if err != nil || string(got) != body {
			t.Fatalf("%s: got %q, %v; want %q", name, got, err, body)
		}
	}
	if _, err := os.Stat(oldDB); !os.IsNotExist(err) {
		t.Fatalf("old database still exists (err = %v)", err)
	}
}

func TestKeepsExistingDatabase(t *testing.T) {
	clearEnv(t)
	data := t.TempDir()
	t.Setenv("XDG_DATA_HOME", data)
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	newDB := filepath.Join(data, "kiipr", "kiipr.db")
	if err := os.MkdirAll(filepath.Dir(newDB), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newDB, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldDB := filepath.Join(data, "suchi", "suchi.db")
	if err := os.MkdirAll(filepath.Dir(oldDB), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldDB, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	c := Default()
	if c.DBPath != newDB {
		t.Fatalf("DBPath = %q, want %q", c.DBPath, newDB)
	}
	if got, err := os.ReadFile(newDB); err != nil || string(got) != "new" {
		t.Fatalf("new database = %q, %v", got, err)
	}
	if _, err := os.Stat(oldDB); err != nil {
		t.Fatalf("old database should be left alone: %v", err)
	}
}

func TestValidate(t *testing.T) {
	clearEnv(t)
	good := Default()
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*Config){
		"backend":  func(c *Config) { c.Backend = "x" },
		"limit":    func(c *Config) { c.Limit = 0 },
		"max-size": func(c *Config) { c.MaxBytes = 0 },
	} {
		c := good
		mutate(&c)
		if err := c.Validate(); err == nil {
			t.Errorf("%s: expected a validation error", name)
		}
	}
}
