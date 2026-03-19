package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.DefaultConnector != "claude" {
		t.Fatalf("expected default connector claude, got %s", cfg.DefaultConnector)
	}
	if cfg.WorkDir == "" {
		t.Fatal("expected non-empty WorkDir")
	}
	if cfg.GlobalMode {
		t.Fatal("expected GlobalMode to be false by default")
	}
}

func TestRegistryDir_Local(t *testing.T) {
	cfg := &Config{WorkDir: "/tmp/myproject"}
	dir := cfg.RegistryDir()
	expected := filepath.Join("/tmp/myproject", ".mdm")
	if dir != expected {
		t.Fatalf("expected %s, got %s", expected, dir)
	}
}

func TestRegistryDir_Global(t *testing.T) {
	cfg := &Config{GlobalMode: true}
	dir := cfg.RegistryDir()
	home, _ := os.UserHomeDir()
	if !strings.HasPrefix(dir, home) {
		t.Fatalf("expected global dir under home, got %s", dir)
	}
	if !strings.HasSuffix(dir, ".mdm") {
		t.Fatalf("expected dir to end with .mdm, got %s", dir)
	}
}
