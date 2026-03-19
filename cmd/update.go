package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

type ghRelease struct {
	TagName string `json:"tag_name"`
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update mdm to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Current version: %s\n", Version)

		resp, err := http.Get("https://api.github.com/repos/Titovilal/middleman/releases/latest")
		if err != nil {
			return fmt.Errorf("failed to check for updates: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return fmt.Errorf("GitHub API returned %d", resp.StatusCode)
		}

		var release ghRelease
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return fmt.Errorf("failed to parse release info: %w", err)
		}

		latest := strings.TrimPrefix(release.TagName, "v")
		current := strings.TrimPrefix(Version, "v")
		if latest == current {
			fmt.Println("Already up to date.")
			return nil
		}

		fmt.Printf("New version available: %s\n", latest)

		goos := runtime.GOOS
		goarch := runtime.GOARCH
		asset := fmt.Sprintf("mdm-%s-%s", goos, goarch)
		if goos == "windows" {
			asset += ".exe"
		}
		dlURL := fmt.Sprintf("https://github.com/Titovilal/middleman/releases/download/%s/%s", latest, asset)

		// Download using net/http (no dependency on curl).
		fmt.Printf("Downloading %s...\n", latest)
		tmp := filepath.Join(os.TempDir(), "mdm-update")
		if err := downloadFile(dlURL, tmp); err != nil {
			return err
		}

		if goos != "windows" {
			if err := os.Chmod(tmp, 0o755); err != nil {
				return fmt.Errorf("chmod failed: %w", err)
			}
		}

		// Determine install target.
		installPath, migrated := resolveInstallPath()
		fmt.Printf("Installing to %s...\n", installPath)

		if err := os.MkdirAll(filepath.Dir(installPath), 0o755); err != nil {
			return fmt.Errorf("create install dir: %w", err)
		}

		if err := installBinary(tmp, installPath); err != nil {
			return err
		}

		// On Windows, ensure user PATH includes the install dir.
		if goos == "windows" {
			ensureWindowsPath(filepath.Dir(installPath))
		}

		if migrated {
			fmt.Printf("Migrated from system directory to %s.\n", installPath)
			if goos == "windows" {
				fmt.Println("Restart your terminal for PATH changes to take effect.")
			}
		}

		fmt.Printf("Updated to %s.\n", latest)
		return nil
	},
}

// resolveInstallPath returns the target binary path and whether a migration happened.
// On Windows: always use %LOCALAPPDATA%\mdm\mdm.exe (user-writable).
// On Unix: use the current binary location, or ~/.local/bin/mdm if in a system dir.
func resolveInstallPath() (string, bool) {
	currentBin, err := os.Executable()
	if err != nil {
		currentBin = "mdm"
	}
	currentBin, _ = filepath.EvalSymlinks(currentBin)

	if runtime.GOOS == "windows" {
		userDir := filepath.Join(os.Getenv("LOCALAPPDATA"), "mdm", "mdm.exe")
		// If currently running from a system dir (e.g. system32), migrate.
		lower := strings.ToLower(currentBin)
		if strings.Contains(lower, "\\windows\\") || strings.Contains(lower, "\\system32\\") {
			return userDir, true
		}
		// If already in user dir, stay there.
		if strings.EqualFold(currentBin, userDir) {
			return userDir, false
		}
		// Otherwise use the user dir too (safer default).
		return userDir, true
	}

	// Unix: if binary is in /usr/ (needs sudo), migrate to ~/.local/bin/
	if strings.HasPrefix(currentBin, "/usr/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local", "bin", "mdm"), true
	}
	return currentBin, false
}

// ensureWindowsPath adds dir to the user PATH if not already present.
func ensureWindowsPath(dir string) {
	psScript := fmt.Sprintf(
		`$p = [Environment]::GetEnvironmentVariable('Path','User'); if ($p -notlike '*%s*') { [Environment]::SetEnvironmentVariable('Path', "$p;%s", 'User') }`,
		dir, dir,
	)
	psCmd := exec.Command("powershell", "-NoProfile", "-Command", psScript)
	_ = psCmd.Run()
}

// downloadFile downloads a URL to a local file using net/http.
func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write download: %w", err)
	}
	return nil
}

// installBinary moves the temp file to the target, with sudo fallback on Unix.
func installBinary(tmp, target string) error {
	if err := os.Rename(tmp, target); err != nil {
		// Rename failed (cross-device or permissions).
		if runtime.GOOS != "windows" && strings.HasPrefix(target, "/usr/") {
			mvCmd := exec.Command("sudo", "mv", tmp, target)
			mvCmd.Stderr = os.Stderr
			if err := mvCmd.Run(); err != nil {
				return fmt.Errorf("install failed (try running with sudo): %w", err)
			}
			return nil
		}
		// Fallback: copy bytes (handles cross-device on Windows too).
		return copyFile(tmp, target)
	}
	return nil
}

// copyFile copies src to dst byte-by-byte (fallback when rename fails).
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create target: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy binary: %w", err)
	}
	_ = os.Remove(src)
	return nil
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the current version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(strings.TrimPrefix(Version, "v"))
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(versionCmd)
}
