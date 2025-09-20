package scanner

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetHomeDir(t *testing.T) {
	home, err := GetHomeDir()
	if err != nil {
		t.Fatalf("GetHomeDir() error = %v", err)
	}

	if home == "" {
		t.Error("GetHomeDir() returned empty string")
	}

	// Verify the home directory exists
	if _, err := os.Stat(home); err != nil {
		t.Errorf("GetHomeDir() returned non-existent directory: %v", home)
	}
}

func TestFileExists(t *testing.T) {
	// Create a temp file for testing
	tempFile, err := os.CreateTemp("", "test-file-exists-*")
	if err != nil {
		t.Fatal(err)
	}
	tempPath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempPath)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "existing file",
			path:     tempPath,
			expected: true,
		},
		{
			name:     "non-existent file",
			path:     "/non/existent/file/that/should/not/exist.txt",
			expected: false,
		},
		{
			name:     "empty path",
			path:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FileExists(tt.path)
			if result != tt.expected {
				t.Errorf("FileExists(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestFindFiles(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "test-find-files-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	files := []string{
		"test1.txt",
		"test2.txt",
		"data.json",
		"config.yaml",
		"subdir/test3.txt",
		"subdir/nested/test4.txt",
		"subdir/nested/data2.json",
	}

	for _, file := range files {
		fullPath := filepath.Join(tempDir, file)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name          string
		root          string
		pattern       string
		expectedCount int
		shouldContain []string
	}{
		{
			name:          "find all txt files",
			root:          tempDir,
			pattern:       "*.txt",
			expectedCount: 4,
			shouldContain: []string{"test1.txt", "test2.txt", "test3.txt", "test4.txt"},
		},
		{
			name:          "find json files",
			root:          tempDir,
			pattern:       "*.json",
			expectedCount: 2,
			shouldContain: []string{"data.json", "data2.json"},
		},
		{
			name:          "find yaml files",
			root:          tempDir,
			pattern:       "*.yaml",
			expectedCount: 1,
			shouldContain: []string{"config.yaml"},
		},
		{
			name:          "pattern with no matches",
			root:          tempDir,
			pattern:       "*.go",
			expectedCount: 0,
			shouldContain: []string{},
		},
		{
			name:          "non-existent root directory",
			root:          "/non/existent/path",
			pattern:       "*.txt",
			expectedCount: 0,
			shouldContain: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := FindFiles(tt.root, tt.pattern)
			if err != nil && tt.root != "/non/existent/path" {
				t.Errorf("FindFiles() error = %v", err)
			}

			if len(found) != tt.expectedCount {
				t.Errorf("FindFiles() found %d files, want %d", len(found), tt.expectedCount)
			}

			for _, expectedFile := range tt.shouldContain {
				fileFound := false
				for _, f := range found {
					if filepath.Base(f) == expectedFile {
						fileFound = true
						break
					}
				}
				if !fileFound {
					t.Errorf("FindFiles() did not find expected file: %s", expectedFile)
				}
			}
		})
	}
}

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		pattern  string
		expected bool
	}{
		{
			name:     "exact match",
			path:     "/path/to/file.txt",
			pattern:  "file.txt",
			expected: true,
		},
		{
			name:     "wildcard match",
			path:     "/path/to/test.json",
			pattern:  "*.json",
			expected: true,
		},
		{
			name:     "no match different extension",
			path:     "/path/to/file.txt",
			pattern:  "*.json",
			expected: false,
		},
		{
			name:     "partial filename match",
			path:     "/path/to/myfile.txt",
			pattern:  "*file.txt",
			expected: true,
		},
		{
			name:     "question mark wildcard",
			path:     "/path/to/file1.txt",
			pattern:  "file?.txt",
			expected: true,
		},
		{
			name:     "no match",
			path:     "/path/to/data.csv",
			pattern:  "*.txt",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesPattern(tt.path, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchesPattern(%q, %q) = %v, want %v", tt.path, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestGetProjectFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "standard project path",
			path:     "/Users/john/projects/my-app/src/main.go",
			expected: "src",
		},
		{
			name:     "hidden directory in path",
			path:     "/Users/john/.config/app/config.json",
			expected: "app",
		},
		{
			name:     "root level file",
			path:     "/file.txt",
			expected: "unknown",
		},
		{
			name:     "empty path",
			path:     "",
			expected: "unknown",
		},
		{
			name:     "path with only hidden directories",
			path:     "/.hidden/.config/.cache/file.txt",
			expected: "unknown",
		},
		{
			name:     "windows-style path",
			path:     "C:\\Users\\john\\projects\\app\\main.go",
			expected: "unknown", // On Unix systems, backslashes aren't path separators
		},
		{
			name:     "relative path",
			path:     "projects/my-app/src/file.go",
			expected: "src",
		},
		{
			name:     "path ending with separator",
			path:     "/home/user/project/",
			expected: "project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetProjectFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("GetProjectFromPath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestFindFiles_SymlinkHandling(t *testing.T) {
	// Create a temporary directory structure with symlinks
	tempDir, err := os.MkdirTemp("", "test-symlinks-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a real file
	realFile := filepath.Join(tempDir, "real.txt")
	if err := os.WriteFile(realFile, []byte("real content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a directory with a file
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	subFile := filepath.Join(subDir, "sub.txt")
	if err := os.WriteFile(subFile, []byte("sub content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink to the file
	symFile := filepath.Join(tempDir, "sym.txt")
	if err := os.Symlink(realFile, symFile); err != nil {
		// Skip test if symlinks aren't supported
		t.Skip("Symlinks not supported on this system")
	}

	// Create a symlink to the directory
	symDir := filepath.Join(tempDir, "symdir")
	if err := os.Symlink(subDir, symDir); err != nil {
		t.Skip("Symlinks not supported on this system")
	}

	// Test finding files with symlinks
	found, err := FindFiles(tempDir, "*.txt")
	if err != nil {
		t.Errorf("FindFiles() error = %v", err)
	}

	// Should find real files and symlinked files
	expectedMin := 2 // At least real.txt and sub.txt
	if len(found) < expectedMin {
		t.Errorf("FindFiles() found %d files, want at least %d", len(found), expectedMin)
	}
}

func TestFindFiles_PermissionDenied(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "test-perms-*")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		// Restore permissions before cleanup
		os.Chmod(filepath.Join(tempDir, "restricted"), 0755)
		os.RemoveAll(tempDir)
	}()

	// Create a restricted directory
	restrictedDir := filepath.Join(tempDir, "restricted")
	if err := os.MkdirAll(restrictedDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file in the restricted directory
	restrictedFile := filepath.Join(restrictedDir, "secret.txt")
	if err := os.WriteFile(restrictedFile, []byte("secret"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create an accessible file
	accessibleFile := filepath.Join(tempDir, "public.txt")
	if err := os.WriteFile(accessibleFile, []byte("public"), 0644); err != nil {
		t.Fatal(err)
	}

	// Remove read permissions from the directory
	if err := os.Chmod(restrictedDir, 0000); err != nil {
		t.Fatal(err)
	}

	// FindFiles should handle permission errors gracefully
	found, err := FindFiles(tempDir, "*.txt")
	if err != nil {
		t.Errorf("FindFiles() should not return error for permission denied: %v", err)
	}

	// Should find the accessible file but not the restricted one
	foundAccessible := false
	for _, f := range found {
		if filepath.Base(f) == "public.txt" {
			foundAccessible = true
		}
		if filepath.Base(f) == "secret.txt" {
			t.Error("FindFiles() should not find files in restricted directories")
		}
	}

	if !foundAccessible {
		t.Error("FindFiles() should find accessible files")
	}
}

func TestSessionInfo(t *testing.T) {
	// Test SessionInfo struct initialization and field access
	now := time.Now().Format(time.RFC3339)
	info := SessionInfo{
		Path:        "/path/to/session.json",
		Tool:        "claude",
		ProjectName: "my-project",
		Size:        1024,
		ModTime:     now,
	}

	if info.Path != "/path/to/session.json" {
		t.Errorf("SessionInfo.Path = %v, want %v", info.Path, "/path/to/session.json")
	}
	if info.Tool != "claude" {
		t.Errorf("SessionInfo.Tool = %v, want %v", info.Tool, "claude")
	}
	if info.ProjectName != "my-project" {
		t.Errorf("SessionInfo.ProjectName = %v, want %v", info.ProjectName, "my-project")
	}
	if info.Size != 1024 {
		t.Errorf("SessionInfo.Size = %v, want %v", info.Size, 1024)
	}
	if info.ModTime != now {
		t.Errorf("SessionInfo.ModTime = %v, want %v", info.ModTime, now)
	}
}

func TestScanResult(t *testing.T) {
	// Test ScanResult struct initialization
	result := ScanResult{
		Tool:          "gpt",
		SessionsFound: 10,
		Imported:      8,
		Failed:        2,
		Errors:        []string{"error1", "error2"},
	}

	if result.Tool != "gpt" {
		t.Errorf("ScanResult.Tool = %v, want %v", result.Tool, "gpt")
	}
	if result.SessionsFound != 10 {
		t.Errorf("ScanResult.SessionsFound = %v, want %v", result.SessionsFound, 10)
	}
	if result.Imported != 8 {
		t.Errorf("ScanResult.Imported = %v, want %v", result.Imported, 8)
	}
	if result.Failed != 2 {
		t.Errorf("ScanResult.Failed = %v, want %v", result.Failed, 2)
	}
	if len(result.Errors) != 2 {
		t.Errorf("ScanResult.Errors length = %v, want %v", len(result.Errors), 2)
	}
}