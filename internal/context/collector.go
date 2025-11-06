package context

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type SystemContext struct {
	WorkingDir string
	Files      []string
	Shell      string
	OS         string
	Arch       string
}

type Collector interface {
	Collect() (*SystemContext, error)
}

type DefaultCollector struct {
	maxFiles int
}

// NewDefaultCollector creates a new collector with file limit
func NewDefaultCollector(maxFiles int) *DefaultCollector {
	return &DefaultCollector{
		maxFiles: maxFiles,
	}
}

// Collect gathers system context information
func (c *DefaultCollector) Collect() (*SystemContext, error) {
	ctx := &SystemContext{}

	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	ctx.WorkingDir = wd

	// Get file listing
	files, _ := c.getFileList() // Don't fail if ls fails
	ctx.Files = files

	// Get shell (default to zsh if not set)
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	ctx.Shell = shell

	// Get OS and architecture
	ctx.OS = runtime.GOOS
	ctx.Arch = runtime.GOARCH

	return ctx, nil
}

// getFileList retrieves a list of files in the current directory
func (c *DefaultCollector) getFileList() ([]string, error) {
	cmd := exec.Command("ls", "-1")
	output, err := cmd.Output()
	if err != nil {
		return []string{}, err
	}

	// Parse output
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Filter out hidden files and limit to maxFiles
	var files []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ".") {
			continue
		}
		files = append(files, line)
		if len(files) >= c.maxFiles {
			break
		}
	}

	return files, nil
}
