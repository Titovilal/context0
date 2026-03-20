package cmd

import (
	"bufio"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var defaultsFS embed.FS

// SetDefaultsFS receives the embedded .mdm/ defaults from main.go.
func SetDefaultsFS(fs embed.FS) { defaultsFS = fs }

var workDir string

var rootCmd = &cobra.Command{
	Use:   "mdm",
	Short: "MDM - AI documentation manager",
	Long:  `MDM manages project documentation in .mdm/ using AI-powered doc generation.`,
	Run: func(cmd *cobra.Command, args []string) {
		printBanner()
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cmd.Name() == "mdm" || cmd.Annotations["skip_init"] == "true" {
			return nil
		}

		if workDir == "" {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			workDir = wd
		}

		guidePath := filepath.Join(workDir, ".mdm", "guides", "the_middleman.md")
		if _, err := os.Stat(guidePath); os.IsNotExist(err) {
			return fmt.Errorf(".mdm/ is not initialized. Run 'mdm init' first")
		}

		return nil
	},
}

var initFlags struct {
	force      bool
	clis       string // comma-separated CLI names, e.g. "claude,gemini"
	defaultCLI string // default CLI name
	syncDocs   bool   // run sync-docs after init
}

// cliIntegration maps each CLI to the extra file it needs (beyond AGENTS.md).
var cliIntegrations = []struct {
	Name      string
	ExtraFile string // empty if only AGENTS.md is needed
}{
	{Name: "claude", ExtraFile: "CLAUDE.md"},
	{Name: "codex", ExtraFile: ""},
	{Name: "copilot", ExtraFile: ""},
	{Name: "gemini", ExtraFile: "GEMINI.md"},
	{Name: "opencode", ExtraFile: ""},
}

var initCmd = &cobra.Command{
	Use:         "init",
	Short:       "Initialize .mdm/ in the current project",
	Annotations: map[string]string{"skip_init": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if workDir == "" {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			workDir = wd
		}

		force := initFlags.force

		// --- .mdm/ directory ---
		mdmDir := filepath.Join(workDir, ".mdm")
		if !force {
			if _, err := os.Stat(mdmDir); !os.IsNotExist(err) {
				if !confirmOverwrite(".mdm/") {
					fmt.Println("Skipped .mdm/")
				} else {
					force = true
				}
			}
		}

		if err := os.MkdirAll(mdmDir, 0o755); err != nil {
			return fmt.Errorf("create .mdm dir: %w", err)
		}

		initDefaults(mdmDir, defaultsFS, force)
		_ = os.MkdirAll(filepath.Join(mdmDir, "docs"), 0o755)

		// --- CLI selection ---
		var selected []struct {
			Name      string
			ExtraFile string
		}
		if initFlags.clis != "" {
			selected = parseCLINames(initFlags.clis)
		} else {
			selected = selectCLIs()
		}

		// AGENTS.md always gets copied (all CLIs need it).
		copyRootFile("AGENTS.md", initFlags.force)

		// Copy extra files for selected CLIs.
		copied := map[string]bool{}
		for _, cli := range selected {
			if cli.ExtraFile != "" && !copied[cli.ExtraFile] {
				copyRootFile(cli.ExtraFile, initFlags.force)
				copied[cli.ExtraFile] = true
			}
		}

		// --- Default CLI ---
		defaultCLI := ""
		if initFlags.defaultCLI != "" {
			defaultCLI = initFlags.defaultCLI
		} else if len(selected) == 1 {
			defaultCLI = selected[0].Name
		} else {
			defaultCLI = selectDefaultCLI(selected)
		}
		saveConfig(mdmDir, defaultCLI)

		fmt.Println()
		fmt.Println("Initialized .mdm/ in", workDir)
		fmt.Printf("Default CLI: %s\n", defaultCLI)

		// --- Sync docs ---
		runSync := initFlags.syncDocs
		if !runSync && initFlags.clis == "" {
			runSync = confirmYesNo("Run sync-docs now?")
		}
		if runSync {
			fmt.Println()
			syncDocsCmd.Flags().Set("connector", defaultCLI)
			return syncDocsCmd.RunE(syncDocsCmd, nil)
		}

		fmt.Println("Run 'mdm sync-docs' to generate documentation.")
		return nil
	},
}

func selectCLIs() []struct {
	Name      string
	ExtraFile string
} {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("Which CLIs do you want to integrate?")
	for i, cli := range cliIntegrations {
		extra := ""
		if cli.ExtraFile != "" {
			extra = fmt.Sprintf(" (+ %s)", cli.ExtraFile)
		}
		fmt.Printf("  %d. %s%s\n", i+1, cli.Name, extra)
	}
	fmt.Println()
	fmt.Print("Enter numbers separated by spaces, or 'all' [all]: ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" || strings.ToLower(input) == "all" {
		return cliIntegrations
	}

	var selected []struct {
		Name      string
		ExtraFile string
	}
	for _, part := range strings.Fields(input) {
		var idx int
		if _, err := fmt.Sscanf(part, "%d", &idx); err == nil && idx >= 1 && idx <= len(cliIntegrations) {
			selected = append(selected, cliIntegrations[idx-1])
		}
	}

	if len(selected) == 0 {
		fmt.Println("No valid selection, using all.")
		return cliIntegrations
	}
	return selected
}

func parseCLINames(input string) []struct {
	Name      string
	ExtraFile string
} {
	names := strings.Split(input, ",")
	var selected []struct {
		Name      string
		ExtraFile string
	}
	for _, name := range names {
		name = strings.TrimSpace(strings.ToLower(name))
		if name == "all" {
			return cliIntegrations
		}
		for _, cli := range cliIntegrations {
			if cli.Name == name {
				selected = append(selected, cli)
				break
			}
		}
	}
	if len(selected) == 0 {
		return cliIntegrations
	}
	return selected
}

func confirmYesNo(prompt string) bool {
	fmt.Printf("%s [y/N] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

func selectDefaultCLI(selected []struct {
	Name      string
	ExtraFile string
}) string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Println("Which CLI should be the default?")
	for i, cli := range selected {
		fmt.Printf("  %d. %s\n", i+1, cli.Name)
	}
	fmt.Printf("\nEnter number [1]: ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return selected[0].Name
	}

	var idx int
	if _, err := fmt.Sscanf(input, "%d", &idx); err == nil && idx >= 1 && idx <= len(selected) {
		return selected[idx-1].Name
	}

	return selected[0].Name
}

type mdmConfig struct {
	DefaultCLI string `json:"default_cli"`
}

func saveConfig(mdmDir string, defaultCLI string) {
	cfg := mdmConfig{DefaultCLI: defaultCLI}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	_ = os.WriteFile(filepath.Join(mdmDir, "config.json"), data, 0o644)
}

func loadConfig(mdmDir string) mdmConfig {
	var cfg mdmConfig
	data, err := os.ReadFile(filepath.Join(mdmDir, "config.json"))
	if err != nil {
		return mdmConfig{DefaultCLI: "claude"}
	}
	_ = json.Unmarshal(data, &cfg)
	if cfg.DefaultCLI == "" {
		cfg.DefaultCLI = "claude"
	}
	return cfg
}

func copyRootFile(name string, force bool) {
	target := filepath.Join(workDir, name)
	data, err := fs.ReadFile(defaultsFS, "defaults/"+name)
	if err != nil {
		return
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		if !force && !confirmOverwrite(name) {
			fmt.Printf("Skipped %s\n", name)
			return
		}
	}
	_ = os.WriteFile(target, data, 0o644)
	fmt.Printf("Created %s\n", name)
}

func confirmOverwrite(name string) bool {
	fmt.Printf("%s already exists. Overwrite? [y/N] ", name)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

// initDefaults walks the embedded defaults/ tree and writes each file
// into mdmDir if it doesn't already exist (or always if force is true).
// AGENTS.md and CLAUDE.md are skipped here — they go to the project root.
func initDefaults(mdmDir string, defaultsFS fs.FS, force bool) {
	_ = fs.WalkDir(defaultsFS, "defaults", func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == "defaults" {
			return nil
		}
		rel := path[len("defaults/"):]

		// Skip root-level files handled separately.
		if rel == "AGENTS.md" || rel == "CLAUDE.md" || rel == "GEMINI.md" {
			return nil
		}

		target := filepath.Join(mdmDir, rel)

		if d.IsDir() {
			_ = os.MkdirAll(target, 0o755)
			return nil
		}
		if d.Name() == ".gitkeep" {
			return nil
		}
		if !force {
			if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
				return nil
			}
		}

		data, readErr := fs.ReadFile(defaultsFS, path)
		if readErr != nil {
			return nil
		}
		_ = os.WriteFile(target, data, 0o644)
		return nil
	})
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&workDir, "workdir", "w", "", "project directory (default: current dir)")
	initCmd.Flags().BoolVarP(&initFlags.force, "force", "f", false, "overwrite existing files without asking")
	initCmd.Flags().StringVar(&initFlags.clis, "clis", "", "comma-separated CLIs to integrate (e.g. claude,gemini,codex)")
	initCmd.Flags().StringVar(&initFlags.defaultCLI, "default", "", "default CLI for sync-docs")
	initCmd.Flags().BoolVar(&initFlags.syncDocs, "sync", false, "run sync-docs after init")
	rootCmd.AddCommand(initCmd)
}

func printBanner() {
	const (
		reset  = "\033[0m"
		bold   = "\033[1m"
		dim    = "\033[2m"
		blue   = "\033[34m"
		white  = "\033[97m"
		green  = "\033[32m"
		red    = "\033[38;5;167m"
		yellow = "\033[38;5;220m"
	)

	fmt.Println()
	fmt.Println(red + bold + "  ███╗   ███╗██████╗ ███╗   ███╗" + reset)
	fmt.Println(red + bold + "  ████╗ ████║██╔══██╗████╗ ████║" + reset)
	fmt.Println(yellow + bold + "  ██╔████╔██║██║  ██║██╔████╔██║" + reset)
	fmt.Println(yellow + bold + "  ██║╚██╔╝██║██║  ██║██║╚██╔╝██║" + reset)
	fmt.Println(red + bold + "  ██║ ╚═╝ ██║██████╔╝██║ ╚═╝ ██║" + reset)
	fmt.Println(red + bold + "  ╚═╝     ╚═╝╚═════╝ ╚═╝     ╚═╝" + reset)
	fmt.Println()
	fmt.Println(white + bold + "  The Middleman" + reset + dim + " — One agent to rule them all" + reset)
	fmt.Println()
	fmt.Printf(dim+"  version "+reset+green+"%s"+reset+dim+"  go "+reset+green+"%s"+reset+dim+"  %s/%s"+reset+"\n",
		Version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Println()
	fmt.Println(white + bold + "  Structure:" + reset)
	fmt.Println(dim + "     .mdm/" + reset)
	fmt.Println(dim + "     ├── guides/      " + reset + dim + "instructions for how MDM operates" + reset)
	fmt.Println(dim + "     ├── templates/   " + reset + dim + "templates for generating docs" + reset)
	fmt.Println(dim + "     └── docs/        " + reset + dim + "generated project documentation" + reset)
	fmt.Println()
	fmt.Println(white + bold + "  Commands:" + reset)
	fmt.Println(dim + "     $ " + reset + white + "mdm init" + reset + dim + "          initialize .mdm/ in your project" + reset)
	fmt.Println(dim + "     $ " + reset + white + "mdm sync-docs" + reset + dim + "     generate/update documentation" + reset)
	fmt.Println(dim + "     $ " + reset + white + "mdm update" + reset + dim + "        self-update to the latest version" + reset)
	fmt.Println(dim + "     $ " + reset + white + "mdm version" + reset + dim + "       print current version" + reset)
	fmt.Println()
	fmt.Println(white + bold + "  Quick start:" + reset)
	fmt.Println(dim + "     1. " + reset + white + "mdm init" + reset + dim + "          in your project root" + reset)
	fmt.Println(dim + "     2. " + reset + white + "mdm sync-docs" + reset + dim + "     to generate documentation" + reset)
	fmt.Println(dim + "     3. docs appear in " + reset + white + ".mdm/docs/" + reset)
	fmt.Println()
	fmt.Println(dim + "  Docs: " + reset + blue + "https://github.com/Titovilal/middleman" + reset)
	fmt.Println()
}
