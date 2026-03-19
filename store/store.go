package store

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/Titovilal/middleman/agent"
)

const registryFileName = "registry.json"

type registryFile struct {
	Version int            `json:"version"`
	Agents  []*agent.Agent `json:"agents"`
}

// Store persists the agent registry to a JSON file using atomic writes.
type Store struct {
	path string
	mu   sync.Mutex
}

// New creates a Store pointing to dir/.mdm/registry.json.
// The directory is created if it doesn't exist.
// defaultsFS should contain the embedded .mdm/ defaults (templates, guides,
// agents.md). Files are written only if they don't already exist.
func New(dir string, defaultsFS fs.FS) (*Store, error) {
	ctmDir := filepath.Join(dir, ".mdm")
	if err := os.MkdirAll(ctmDir, 0o755); err != nil {
		return nil, fmt.Errorf("create .mdm dir: %w", err)
	}

	// Initialize default files (templates, guides, agents.md) inside .mdm/.
	if defaultsFS != nil {
		initDefaults(ctmDir, defaultsFS)
	}

	return &Store{path: filepath.Join(ctmDir, registryFileName)}, nil
}

// initDefaults walks the embedded defaults/ tree and writes each file
// into mdmDir if it doesn't already exist. The directory structure inside
// defaults/ mirrors what should appear inside .mdm/ (e.g. defaults/templates/
// → .mdm/templates/).
func initDefaults(mdmDir string) {
	_ = fs.WalkDir(defaultsFS, "defaults", func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == "defaults" {
			return nil
		}
		// Strip the "defaults/" prefix to get the relative path inside .mdm/.
		rel := path[len("defaults/"):]
		target := filepath.Join(mdmDir, rel)

		if d.IsDir() {
			_ = os.MkdirAll(target, 0o755)
			return nil
		}

		// Skip .gitkeep markers used only for embedding empty dirs.
		if d.Name() == ".gitkeep" {
			return nil
		}

		// Only write if the file doesn't already exist.
		if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
			return nil
		}

		data, readErr := defaultsFS.ReadFile(path)
		if readErr != nil {
			return nil
		}
		_ = os.WriteFile(target, data, 0o644)
		return nil
	})
}

// NewGlobal creates a Store in ~/.mdm/registry.json.
func NewGlobal() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return New(home)
}

func (s *Store) Load() (*agent.Registry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

func (s *Store) load() (*agent.Registry, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return agent.NewRegistry(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}

	var rf registryFile
	if err := json.Unmarshal(data, &rf); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}

	reg := agent.NewRegistry()
	reg.Agents = rf.Agents
	if reg.Agents == nil {
		reg.Agents = make([]*agent.Agent, 0)
	}
	return reg, nil
}

func (s *Store) Save(r *agent.Registry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save(r)
}

func (s *Store) save(r *agent.Registry) error {
	rf := registryFile{
		Version: 1,
		Agents:  r.Agents,
	}
	data, err := json.MarshalIndent(rf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}

	// Atomic write: write to temp file, then rename.
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write temp registry: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename registry: %w", err)
	}
	return nil
}

// WithLock loads the registry, calls fn, then saves. All under the mutex.
func (s *Store) WithLock(fn func(*agent.Registry) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	reg, err := s.load()
	if err != nil {
		return err
	}
	if err := fn(reg); err != nil {
		return err
	}
	return s.save(reg)
}

// Path returns the absolute path to the registry file.
func (s *Store) Path() string { return s.path }
