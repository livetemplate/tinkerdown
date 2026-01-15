package commands

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// maxDirTraversalDepth is the maximum number of parent directories to search
// when looking for the tinkerdown module's go.mod file.
const maxDirTraversalDepth = 15

// BuildCommand implements the build command.
// It compiles a tinkerdown app into a standalone executable.
func BuildCommand(args []string) error {
	// Parse arguments
	var inputPath string
	var outputPath string
	var target string

	// Parse flags
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--output" || arg == "-o" {
			if i+1 < len(args) {
				outputPath = args[i+1]
				i++
			}
		} else if val, ok := strings.CutPrefix(arg, "--output="); ok {
			outputPath = val
		} else if val, ok := strings.CutPrefix(arg, "-o="); ok {
			outputPath = val
		} else if arg == "--target" || arg == "-t" {
			if i+1 < len(args) {
				target = args[i+1]
				i++
			}
		} else if val, ok := strings.CutPrefix(arg, "--target="); ok {
			target = val
		} else if val, ok := strings.CutPrefix(arg, "-t="); ok {
			target = val
		} else if !strings.HasPrefix(arg, "-") {
			// Positional argument (input path)
			inputPath = arg
		}
	}

	// Validate input
	if inputPath == "" {
		return fmt.Errorf("input path required\n\nUsage: tinkerdown build <file.md|directory> [--output=<binary>] [--target=<os/arch>]\n\nExamples:\n  tinkerdown build app.md -o myapp\n  tinkerdown build ./docs -o docs-server\n  tinkerdown build app.md --target=linux/amd64 -o myapp-linux")
	}

	// Check if input exists
	info, err := os.Stat(inputPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("input path does not exist: %s", inputPath)
		}
		return fmt.Errorf("failed to stat input: %w", err)
	}

	// Set default output name
	if outputPath == "" {
		if info.IsDir() {
			// Handle special paths like "." or "/"
			base := filepath.Base(filepath.Clean(inputPath))
			if base == "." || base == string(os.PathSeparator) {
				// Use current directory's actual name
				if wd, err := os.Getwd(); err == nil {
					if wdBase := filepath.Base(wd); wdBase != "" && wdBase != "." && wdBase != string(os.PathSeparator) {
						base = wdBase
					} else {
						base = "server"
					}
				} else {
					base = "server"
				}
			}
			outputPath = base + "-server"
		} else {
			// Handle edge case where input is just ".md"
			base := filepath.Base(inputPath)
			name := strings.TrimSuffix(base, ".md")
			if name == "" {
				return fmt.Errorf("cannot derive default output name from input file %q; please specify --output", inputPath)
			}
			outputPath = name
		}
	}

	// Get absolute paths
	absInput, err := filepath.Abs(inputPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	absOutput, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute output path: %w", err)
	}

	fmt.Printf("🔨 Building tinkerdown app...\n")
	fmt.Printf("   Input: %s\n", absInput)
	fmt.Printf("   Output: %s\n", absOutput)
	if target != "" {
		fmt.Printf("   Target: %s\n", target)
	}

	// Generate build source
	tmpDir, err := generateBuildSource(absInput, info.IsDir())
	if err != nil {
		return fmt.Errorf("failed to generate build source: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Build the binary
	if err := buildBinary(tmpDir, absOutput, target); err != nil {
		return fmt.Errorf("failed to build binary: %w", err)
	}

	fmt.Printf("\n✅ Build successful!\n")
	fmt.Printf("   Run with: ./%s\n", filepath.Base(absOutput))
	fmt.Printf("   Options:  --port=8080 --host=localhost\n")

	return nil
}

// generateBuildSource creates a temporary directory with Go source code
// that embeds the markdown content and serves it.
func generateBuildSource(inputPath string, isDir bool) (string, error) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "tinkerdown-build-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Create content directory for embedded files
	contentDir := filepath.Join(tmpDir, "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to create content directory: %w", err)
	}

	// Copy markdown files to content directory
	if isDir {
		if err := copyDirectory(inputPath, contentDir); err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("failed to copy directory: %w", err)
		}
	} else {
		// Single file - copy to content/app.md
		content, err := os.ReadFile(inputPath)
		if err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("failed to read input file: %w", err)
		}
		if err := os.WriteFile(filepath.Join(contentDir, "app.md"), content, 0644); err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("failed to write content file: %w", err)
		}
	}

	// Copy config file if it exists (aligned with config.LoadFromDir search order)
	configFiles := []string{"tinkerdown.yaml", "lmt.yaml", "livemdtools.yaml"}
	for _, configFile := range configFiles {
		var configPath string
		if isDir {
			configPath = filepath.Join(inputPath, configFile)
		} else {
			configPath = filepath.Join(filepath.Dir(inputPath), configFile)
		}
		if _, err := os.Stat(configPath); err == nil {
			configContent, err := os.ReadFile(configPath)
			if err != nil {
				continue // Try next config file
			}
			if err := os.WriteFile(filepath.Join(contentDir, configFile), configContent, 0644); err != nil {
				os.RemoveAll(tmpDir)
				return "", fmt.Errorf("failed to write config file: %w", err)
			}
			break
		}
	}

	// Generate main.go
	mainGo := generateMainGo()
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to write main.go: %w", err)
	}

	// Find tinkerdown module path for go.mod
	tinkerdownPath, tinkerdownVersion, err := findTinkerdownModule()
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to find tinkerdown module: %w", err)
	}

	// Generate go.mod
	var goMod string
	if tinkerdownPath != "" {
		// Use replace directive for local development
		goMod = fmt.Sprintf(`module tinkerdown-app

go 1.22

require github.com/livetemplate/tinkerdown v0.0.0

replace github.com/livetemplate/tinkerdown => %s
`, tinkerdownPath)
	} else {
		// Use specific version from module cache
		goMod = fmt.Sprintf(`module tinkerdown-app

go 1.22

require github.com/livetemplate/tinkerdown %s
`, tinkerdownVersion)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("failed to write go.mod: %w", err)
	}

	return tmpDir, nil
}

// tinkerdownSourcePath is set at compile time via ldflags for development builds.
// This allows the build command to find the source even when running from temp directories.
var tinkerdownSourcePath string

// findTinkerdownModule finds the tinkerdown module for use in generated go.mod.
// Returns (localPath, version, error). If localPath is non-empty, use replace directive.
func findTinkerdownModule() (string, string, error) {
	// First, check if path was set at compile time
	if tinkerdownSourcePath != "" {
		if _, err := os.Stat(filepath.Join(tinkerdownSourcePath, "go.mod")); err == nil {
			return tinkerdownSourcePath, "", nil
		}
	}

	// Try to find from the executable's location by resolving symlinks
	execPath, err := os.Executable()
	if err == nil {
		// Resolve any symlinks
		realPath, err := filepath.EvalSymlinks(execPath)
		if err == nil {
			execPath = realPath
		}

		if dir := findModuleInParents(filepath.Dir(execPath)); dir != "" {
			return dir, "", nil
		}
	}

	// Try to find from working directory (for development)
	if wd, err := os.Getwd(); err == nil {
		if dir := findModuleInParents(wd); dir != "" {
			return dir, "", nil
		}
	}

	// Try to find from GOPATH/pkg/mod
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		homeDir, _ := os.UserHomeDir()
		gopath = filepath.Join(homeDir, "go")
	}

	modCache := filepath.Join(gopath, "pkg", "mod", "github.com", "livetemplate")
	entries, err := os.ReadDir(modCache)
	if err == nil {
		// Find the latest tinkerdown version using semver-aware comparison
		var latestVersion string
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "tinkerdown@") {
				version := strings.TrimPrefix(entry.Name(), "tinkerdown@")
				if latestVersion == "" || compareSemver(version, latestVersion) > 0 {
					latestVersion = version
				}
			}
		}
		if latestVersion != "" {
			return "", latestVersion, nil
		}
	}

	return "", "", fmt.Errorf("could not find tinkerdown module - please ensure tinkerdown is installed or run from source directory")
}

// findModuleInParents walks up the directory tree looking for the tinkerdown module's go.mod.
func findModuleInParents(startDir string) string {
	dir := startDir
	for i := 0; i < maxDirTraversalDepth; i++ {
		goModPath := filepath.Join(dir, "go.mod")
		if content, err := os.ReadFile(goModPath); err == nil {
			if strings.Contains(string(content), "module github.com/livetemplate/tinkerdown") {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// compareSemver compares two semantic version strings.
// Returns positive if v1 > v2, negative if v1 < v2, zero if equal.
// Handles versions like "v0.9.0" vs "v0.10.0" correctly.
func compareSemver(v1, v2 string) int {
	// Strip "v" prefix if present
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Compare each part numerically
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(parts1) {
			// Parse numeric part (ignore pre-release suffixes)
			numStr := parts1[i]
			if idx := strings.IndexAny(numStr, "-+"); idx >= 0 {
				numStr = numStr[:idx]
			}
			fmt.Sscanf(numStr, "%d", &n1)
		}
		if i < len(parts2) {
			numStr := parts2[i]
			if idx := strings.IndexAny(numStr, "-+"); idx >= 0 {
				numStr = numStr[:idx]
			}
			fmt.Sscanf(numStr, "%d", &n2)
		}
		if n1 != n2 {
			return n1 - n2
		}
	}
	return 0
}

// generateMainGo generates the main.go source code for the standalone binary.
func generateMainGo() string {
	return `package main

import (
	"embed"
	"flag"
	"fmt"
	"os"

	"github.com/livetemplate/tinkerdown/pkg/embedded"
)

//go:embed content/*
var contentFS embed.FS

func main() {
	port := flag.Int("port", 8080, "Server port")
	host := flag.String("host", "localhost", "Server host")
	flag.Parse()

	addr := fmt.Sprintf("%s:%d", *host, *port)
	fmt.Printf("📚 Tinkerdown Server\n\n")
	fmt.Printf("🌐 Server running at http://%s\n", addr)
	fmt.Printf("Press Ctrl+C to stop\n\n")

	if err := embedded.Serve(contentFS, "content", addr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
`
}

// copyDirectory recursively copies a directory, filtering for relevant files.
func copyDirectory(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Skip hidden directories and files (except config files)
		if strings.HasPrefix(d.Name(), ".") && !strings.HasPrefix(d.Name(), ".tinkerdown") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories starting with _
		if d.IsDir() && strings.HasPrefix(d.Name(), "_") {
			return filepath.SkipDir
		}

		targetPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		// Only copy relevant files (.md, .yaml, .yml, .json, .csv, .db)
		ext := strings.ToLower(filepath.Ext(d.Name()))
		validExts := map[string]bool{
			".md": true, ".yaml": true, ".yml": true,
			".json": true, ".csv": true, ".db": true,
		}
		if !validExts[ext] {
			return nil
		}

		// Copy file
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, content, 0644)
	})
}

// buildBinary runs go build to create the executable.
func buildBinary(srcDir, outputPath, target string) error {
	// Prepare build command
	args := []string{"build", "-o", outputPath, "."}
	cmd := exec.Command("go", args...)
	cmd.Dir = srcDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up environment
	env := os.Environ()

	// Handle cross-compilation target
	if target != "" {
		parts := strings.Split(target, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid target format %q, expected os/arch (e.g., linux/amd64)", target)
		}
		goos := parts[0]
		goarch := parts[1]

		// Validate GOOS
		validGOOS := map[string]bool{
			"linux": true, "darwin": true, "windows": true,
			"freebsd": true, "openbsd": true, "netbsd": true,
		}
		if !validGOOS[goos] {
			return fmt.Errorf("unsupported GOOS: %s", goos)
		}

		// Validate GOARCH
		validGOARCH := map[string]bool{
			"amd64": true, "arm64": true, "386": true, "arm": true,
		}
		if !validGOARCH[goarch] {
			return fmt.Errorf("unsupported GOARCH: %s", goarch)
		}

		env = append(env, "GOOS="+goos, "GOARCH="+goarch)
		// Disable CGO for cross-compilation
		env = append(env, "CGO_ENABLED=0")
	}

	cmd.Env = env

	// Run go mod tidy first to resolve dependencies
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = srcDir
	tidyCmd.Env = env
	tidyCmd.Stdout = os.Stdout
	tidyCmd.Stderr = os.Stderr
	if err := tidyCmd.Run(); err != nil {
		return fmt.Errorf("go mod tidy failed: %w", err)
	}

	// Build the binary
	fmt.Printf("\n🔧 Compiling...\n")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	return nil
}
