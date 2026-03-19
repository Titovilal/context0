package cmd

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// sourceExtensions are file extensions considered as source code.
var sourceExtensions = map[string]bool{
	".go": true, ".py": true, ".js": true, ".ts": true, ".tsx": true, ".jsx": true,
	".java": true, ".kt": true, ".rs": true, ".c": true, ".cpp": true, ".h": true,
	".cs": true, ".rb": true, ".php": true, ".swift": true, ".scala": true,
	".lua": true, ".sh": true, ".bash": true, ".zsh": true, ".pl": true,
	".r": true, ".R": true, ".m": true, ".mm": true, ".zig": true, ".nim": true,
	".ex": true, ".exs": true, ".erl": true, ".hs": true, ".ml": true,
	".vue": true, ".svelte": true, ".dart": true, ".proto": true,
}

// skipDirs are directories to skip when scanning.
var skipDirs = map[string]bool{
	".git": true, ".mdm": true, "node_modules": true, "vendor": true,
	".venv": true, "venv": true, "__pycache__": true, ".tox": true,
	"dist": true, "build": true, "target": true, "bin": true, "obj": true,
	".next": true, ".nuxt": true, ".cache": true, ".idea": true, ".vscode": true,
}

// docGroup represents a group of files that will become one doc.
type docGroup struct {
	Name  string   // doc filename without .md (e.g. "cmd" or "connector")
	Title string   // human-readable heading
	Files []string // relative paths from project root
}

var syncDocsFlags struct {
	dryRun bool
}

var syncDocsCmd = &cobra.Command{
	Use:   "sync-docs",
	Short: "Create or update feature documentation in .mdm/docs/",
	Long: `Scans the project source files and generates documentation in .mdm/docs/.
Files are grouped by directory into docs of 8-16 files each, following the doc template.
Existing docs are updated only if the file list has changed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		wd := flags.workDir
		if wd == "" {
			var err error
			wd, err = os.Getwd()
			if err != nil {
				return err
			}
		}

		docsDir := filepath.Join(wd, ".mdm", "docs")
		if err := os.MkdirAll(docsDir, 0o755); err != nil {
			return fmt.Errorf("create docs dir: %w", err)
		}

		// Scan project for source files.
		files, err := scanSourceFiles(wd)
		if err != nil {
			return fmt.Errorf("scan files: %w", err)
		}
		if len(files) == 0 {
			fmt.Println("no source files found")
			return nil
		}

		// Group files by top-level directory, then merge small groups / split large ones.
		groups := groupFiles(files)

		var created, updated, unchanged int

		for _, g := range groups {
			content := renderGroupDoc(g)
			docPath := filepath.Join(docsDir, g.Name+".md")

			existing, err := os.ReadFile(docPath)
			if err == nil {
				if hashContent(existing) == hashContent([]byte(content)) {
					unchanged++
					continue
				}
				if syncDocsFlags.dryRun {
					fmt.Printf("  update  %s.md\n", g.Name)
				} else {
					if err := os.WriteFile(docPath, []byte(content), 0o644); err != nil {
						return fmt.Errorf("write %s: %w", g.Name, err)
					}
					fmt.Printf("  updated  %s.md\n", g.Name)
				}
				updated++
			} else {
				if syncDocsFlags.dryRun {
					fmt.Printf("  create  %s.md\n", g.Name)
				} else {
					if err := os.WriteFile(docPath, []byte(content), 0o644); err != nil {
						return fmt.Errorf("write %s: %w", g.Name, err)
					}
					fmt.Printf("  created  %s.md\n", g.Name)
				}
				created++
			}
		}

		// Generate project overview.
		overviewContent := renderOverview(wd, groups)
		overviewPath := filepath.Join(docsDir, "doc_project_overview.md")

		existingOverview, err := os.ReadFile(overviewPath)
		if err == nil && hashContent(existingOverview) == hashContent([]byte(overviewContent)) {
			// unchanged
		} else {
			if syncDocsFlags.dryRun {
				if err == nil {
					fmt.Printf("  update  doc_project_overview.md\n")
				} else {
					fmt.Printf("  create  doc_project_overview.md\n")
				}
			} else {
				if err := os.WriteFile(overviewPath, []byte(overviewContent), 0o644); err != nil {
					return fmt.Errorf("write project overview: %w", err)
				}
				if err == nil {
					fmt.Printf("  updated  doc_project_overview.md\n")
				} else {
					fmt.Printf("  created  doc_project_overview.md\n")
				}
			}
			if err == nil {
				updated++
			} else {
				created++
			}
		}

		prefix := ""
		if syncDocsFlags.dryRun {
			prefix = "(dry-run) "
		}
		fmt.Printf("\n%s%d created, %d updated, %d unchanged\n", prefix, created, updated, unchanged)
		return nil
	},
}

// scanSourceFiles walks the project directory and returns relative paths of source files.
func scanSourceFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable
		}
		if info.IsDir() {
			name := info.Name()
			if skipDirs[name] || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if sourceExtensions[ext] {
			rel, _ := filepath.Rel(root, path)
			files = append(files, rel)
		}
		return nil
	})
	return files, err
}

// groupFiles groups files by their top-level directory, then merges/splits
// to keep each group between 8 and 16 files.
func groupFiles(files []string) []docGroup {
	// Bucket by top-level dir (or "root" for files at project root).
	buckets := make(map[string][]string)
	var bucketOrder []string

	for _, f := range files {
		parts := strings.SplitN(f, string(os.PathSeparator), 2)
		key := "root"
		if len(parts) > 1 {
			key = parts[0]
		}
		if _, exists := buckets[key]; !exists {
			bucketOrder = append(bucketOrder, key)
		}
		buckets[key] = append(buckets[key], f)
	}

	sort.Strings(bucketOrder)

	// Build initial groups from buckets.
	var groups []rawGroup
	for _, k := range bucketOrder {
		groups = append(groups, rawGroup{keys: []string{k}, files: buckets[k]})
	}

	// Merge: repeatedly find the smallest group and merge it with its smallest neighbor,
	// until all groups have >= 8 files or only one group remains.
	for len(groups) > 1 {
		// Find smallest group.
		minIdx := 0
		for i := range groups {
			if len(groups[i].files) < len(groups[minIdx].files) {
				minIdx = i
			}
		}
		if len(groups[minIdx].files) >= 8 {
			break // all groups are big enough
		}

		// Pick the smaller neighbor to merge with.
		mergeWith := -1
		if minIdx > 0 && minIdx < len(groups)-1 {
			if len(groups[minIdx-1].files) <= len(groups[minIdx+1].files) {
				mergeWith = minIdx - 1
			} else {
				mergeWith = minIdx + 1
			}
		} else if minIdx > 0 {
			mergeWith = minIdx - 1
		} else {
			mergeWith = minIdx + 1
		}

		// Merge minIdx into mergeWith.
		lo, hi := mergeWith, minIdx
		if lo > hi {
			lo, hi = hi, lo
		}
		merged := rawGroup{
			keys:  append(append([]string{}, groups[lo].keys...), groups[hi].keys...),
			files: append(append([]string{}, groups[lo].files...), groups[hi].files...),
		}
		// Replace lo with merged, remove hi.
		groups[lo] = merged
		groups = append(groups[:hi], groups[hi+1:]...)
	}

	// Split large groups (> 16 files).
	var final []docGroup
	for _, rg := range groups {
		sort.Strings(rg.files)
		if len(rg.files) <= 16 {
			final = append(final, docGroup{
				Name:  groupName(rg.keys),
				Title: groupTitle(rg.keys),
				Files: rg.files,
			})
		} else {
			// Split into chunks of ~12.
			chunks := splitSlice(rg.files, 12)
			for i, chunk := range chunks {
				suffix := ""
				if len(chunks) > 1 {
					suffix = fmt.Sprintf("_%d", i+1)
				}
				final = append(final, docGroup{
					Name:  groupName(rg.keys) + suffix,
					Title: groupTitle(rg.keys) + suffix,
					Files: chunk,
				})
			}
		}
	}

	return final
}

type rawGroup struct {
	keys  []string
	files []string
}

func splitSlice(files []string, chunkSize int) [][]string {
	var chunks [][]string
	for i := 0; i < len(files); i += chunkSize {
		end := i + chunkSize
		if end > len(files) {
			end = len(files)
		}
		chunks = append(chunks, files[i:end])
	}
	// If last chunk is under 8, merge it with the previous one.
	if len(chunks) > 1 && len(chunks[len(chunks)-1]) < 8 {
		prev := chunks[len(chunks)-2]
		last := chunks[len(chunks)-1]
		chunks = chunks[:len(chunks)-2]
		chunks = append(chunks, append(prev, last...))
	}
	return chunks
}

func groupName(keys []string) string {
	var parts []string
	for _, k := range keys {
		parts = append(parts, strings.ReplaceAll(k, string(os.PathSeparator), "_"))
	}
	return strings.Join(parts, "_")
}

func groupTitle(keys []string) string {
	var parts []string
	for _, k := range keys {
		if k == "root" {
			parts = append(parts, "Root")
		} else {
			parts = append(parts, k)
		}
	}
	return strings.Join(parts, " + ")
}

// renderGroupDoc renders a doc following the _doc_template structure.
func renderGroupDoc(g docGroup) string {
	var b strings.Builder
	b.WriteString("# " + g.Title + "\n\n")
	b.WriteString("## What It Does\n")
	b.WriteString("<!-- describe what this group of files does -->\n\n")
	b.WriteString("## Main Files\n")
	for _, f := range g.Files {
		b.WriteString(fmt.Sprintf("- `%s`\n", f))
	}
	b.WriteString("\n## Flow\n")
	b.WriteString("<!-- describe how these files work together -->\n")
	return b.String()
}

// renderOverview generates the doc_project_overview.md for the project.
func renderOverview(wd string, groups []docGroup) string {
	projectName := filepath.Base(wd)

	var b strings.Builder
	b.WriteString("# " + projectName + "\n\n")
	b.WriteString("## What It Does\n")
	b.WriteString("<!-- describe the project -->\n\n")

	b.WriteString("## Main Files\n")
	// List top-level directories/files.
	entries, _ := os.ReadDir(wd)
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || skipDirs[name] {
			continue
		}
		if e.IsDir() {
			b.WriteString(fmt.Sprintf("- `%s/`\n", name))
		} else {
			ext := strings.ToLower(filepath.Ext(name))
			if sourceExtensions[ext] || name == "go.mod" || name == "package.json" || name == "Cargo.toml" || name == "Makefile" || name == "Dockerfile" {
				b.WriteString(fmt.Sprintf("- `%s`\n", name))
			}
		}
	}

	b.WriteString("\n## Flow\n")
	b.WriteString("<!-- describe the main usage flow -->\n\n")

	b.WriteString("## Documentation available in `.mdm/docs/`\n\n")
	b.WriteString("- **`_doc_template.md`** — template for creating new docs\n")

	sorted := make([]docGroup, len(groups))
	copy(sorted, groups)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	for _, g := range sorted {
		b.WriteString(fmt.Sprintf("- **`%s.md`** — %s (%d files)\n", g.Name, g.Title, len(g.Files)))
	}

	return b.String()
}

func hashContent(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}

func init() {
	syncDocsCmd.Flags().BoolVar(&syncDocsFlags.dryRun, "dry-run", false, "show what would change without writing files")
	rootCmd.AddCommand(syncDocsCmd)
}
