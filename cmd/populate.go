package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var populateFlags struct {
	force bool // overwrite existing files instead of only adding new ones
}

var populateCmd = &cobra.Command{
	Use:         "populate",
	Short:       "Add new default guides/templates to an existing .ctx/",
	Long:        "Copies default guides and templates into the current .ctx/ directory. By default only missing files are added; use --force to overwrite existing ones.",
	Annotations: map[string]string{"skip_init": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if workDir == "" {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			workDir = wd
		}

		ctxDir := filepath.Join(workDir, ".ctx")
		if _, err := os.Stat(ctxDir); os.IsNotExist(err) {
			return fmt.Errorf(".ctx/ not found in %s. Run 'ctx init' first", workDir)
		}

		added, updated := populateDefaults(ctxDir, defaultsFS, populateFlags.force)

		if added == 0 && updated == 0 {
			stSkip("Nothing to do — .ctx/ already has all default files")
			return nil
		}

		stTitle("Done")
		if added > 0 {
			stDone(fmt.Sprintf("Added %s new file(s)", stValue(fmt.Sprintf("%d", added))))
		}
		if updated > 0 {
			stDone(fmt.Sprintf("Overwrote %s existing file(s)", stValue(fmt.Sprintf("%d", updated))))
		}
		return nil
	},
}

// populateDefaults walks the embedded defaults/ tree and writes each guide/template
// into ctxDir. Missing files are always added; existing files are only overwritten
// when force is true. Root-level files (AGENTS.md, CLAUDE.md, GEMINI.md) are skipped
// since they depend on the user's CLI selection. Returns (added, updated) counts.
func populateDefaults(ctxDir string, defaultsFS fs.FS, force bool) (int, int) {
	added, updated := 0, 0

	_ = fs.WalkDir(defaultsFS, "defaults", func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == "defaults" {
			return nil
		}
		rel := path[len("defaults/"):]

		// Skip root-level files handled separately by init.
		if rel == "AGENTS.md" || rel == "CLAUDE.md" || rel == "GEMINI.md" {
			return nil
		}

		target := filepath.Join(ctxDir, rel)

		if d.IsDir() {
			_ = os.MkdirAll(target, 0o755)
			return nil
		}
		if d.Name() == ".gitkeep" {
			return nil
		}

		exists := false
		if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
			exists = true
		}
		if exists && !force {
			return nil
		}

		data, readErr := fs.ReadFile(defaultsFS, path)
		if readErr != nil {
			return nil
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			return nil
		}

		if exists {
			updated++
			stStep("Overwrote " + stValue(rel))
		} else {
			added++
			stDone("Added " + stValue(rel))
		}
		return nil
	})

	return added, updated
}

func init() {
	populateCmd.Flags().BoolVarP(&populateFlags.force, "force", "f", false, "overwrite existing files instead of only adding missing ones")
	rootCmd.AddCommand(populateCmd)
}
