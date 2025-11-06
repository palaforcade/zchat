package context

import (
	"os"
	"runtime"
	"testing"
)

func TestNewDefaultCollector(t *testing.T) {
	collector := NewDefaultCollector(10)

	if collector.maxFiles != 10 {
		t.Errorf("Expected maxFiles 10, got %d", collector.maxFiles)
	}
}

func TestCollect_SystemInfo(t *testing.T) {
	collector := NewDefaultCollector(20)
	ctx, err := collector.Collect()

	if err != nil {
		t.Fatalf("Collect() failed: %v", err)
	}

	// Check OS and Arch
	if ctx.OS != runtime.GOOS {
		t.Errorf("Expected OS %s, got %s", runtime.GOOS, ctx.OS)
	}

	if ctx.Arch != runtime.GOARCH {
		t.Errorf("Expected Arch %s, got %s", runtime.GOARCH, ctx.Arch)
	}

	// Check WorkingDir is set
	if ctx.WorkingDir == "" {
		t.Error("Expected WorkingDir to be set")
	}

	// Check Shell is set (should have default if not in env)
	if ctx.Shell == "" {
		t.Error("Expected Shell to be set")
	}
}

func TestCollect_FileList(t *testing.T) {
	collector := NewDefaultCollector(5)
	ctx, err := collector.Collect()

	if err != nil {
		t.Fatalf("Collect() failed: %v", err)
	}

	// Files list should exist (may be empty in some test environments)
	if ctx.Files == nil {
		t.Error("Expected Files list to be initialized")
	}

	// If we got files, check they don't exceed maxFiles
	if len(ctx.Files) > 5 {
		t.Errorf("Expected at most 5 files, got %d", len(ctx.Files))
	}

	// Check no hidden files (starting with .)
	for _, file := range ctx.Files {
		if len(file) > 0 && file[0] == '.' {
			t.Errorf("Found hidden file in list: %s", file)
		}
	}
}

func TestGetFileList_MaxFilesLimit(t *testing.T) {
	collector := NewDefaultCollector(3)
	files, err := collector.getFileList()

	// err is ok if ls fails in test environment
	if err == nil {
		if len(files) > 3 {
			t.Errorf("Expected at most 3 files, got %d", len(files))
		}
	}
}

func TestGetFileList_HiddenFilesFiltered(t *testing.T) {
	collector := NewDefaultCollector(100)
	files, _ := collector.getFileList()

	for _, file := range files {
		if len(file) > 0 && file[0] == '.' {
			t.Errorf("Hidden file should be filtered out: %s", file)
		}
	}
}

func TestCollect_ShellDefault(t *testing.T) {
	// Clear SHELL env var temporarily
	originalShell := ""
	if val, exists := os.LookupEnv("SHELL"); exists {
		originalShell = val
		os.Unsetenv("SHELL")
		defer os.Setenv("SHELL", originalShell)
	}

	collector := NewDefaultCollector(20)
	ctx, err := collector.Collect()

	if err != nil {
		t.Fatalf("Collect() failed: %v", err)
	}

	// Should default to /bin/zsh
	if ctx.Shell != "/bin/zsh" {
		t.Errorf("Expected default shell '/bin/zsh', got '%s'", ctx.Shell)
	}
}
