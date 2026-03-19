package store

import (
	"encoding/json"
	"fmt"
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
// It also initializes the __docs__/ folder with default files if they don't exist.
func New(dir string) (*Store, error) {
	ctmDir := filepath.Join(dir, ".mdm")
	if err := os.MkdirAll(ctmDir, 0o755); err != nil {
		return nil, fmt.Errorf("create .mdm dir: %w", err)
	}

	// Initialize docs/ and agents.md inside .mdm/.
	initDocs(ctmDir)
	initAgentsMD(ctmDir)

	return &Store{path: filepath.Join(ctmDir, registryFileName)}, nil
}

// initDocs creates the docs/ folder with the default template inside .mdm/.
func initDocs(mdmDir string) {
	docsDir := filepath.Join(mdmDir, "docs")
	_ = os.MkdirAll(docsDir, 0o755)

	templatePath := filepath.Join(docsDir, "_doc_template.md")
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		_ = os.WriteFile(templatePath, []byte(docTemplate), 0o644)
	}
}

// initAgentsMD creates the agents.md file inside .mdm/.
func initAgentsMD(mdmDir string) {
	agentsPath := filepath.Join(mdmDir, "agents.md")
	if _, err := os.Stat(agentsPath); os.IsNotExist(err) {
		_ = os.WriteFile(agentsPath, []byte(agentsMDContent), 0o644)
	}
}

const docTemplate = `# Guide to Creating Feature Documentation

## Objective
Create simple, high-level documentation that explains what a feature does and how components work together.

## Template

` + "```markdown" + `
# [Feature Name]

## What It Does
[Brief explanation in 1-3 sentences]

## Main Files
- ` + "`file1.py`" + ` - [role]
- ` + "`file2.py`" + ` - [role]

## Flow
1. [What starts the process]
2. [What happens in the middle]
3. [What's the result]
` + "```" + `

## Best Practices

### Do
- Keep it short and simple
- Focus on the big picture
- Use plain language

### Avoid
- Code snippets
- Implementation details
- Complex technical jargon

## File Naming

` + "`[some_name].md`" + `

Example:
- ` + "`project_overview.md`" + `
`

const agentsMDContent = `# Mandatory steps to update the code base

1. List ` + "`.mdm/docs/`" + ` folder.
2. Read ` + "`.mdm/docs/doc_project_overview.md`" + `.
3. Read the next specific document(s) if needed.
4. Make the necessary changes.
5. Update the affected document(s).
6. For more follow-up changes, repeat from step 3.
`

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
