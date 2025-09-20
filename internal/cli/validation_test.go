package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidator_ValidateTool(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name       string
		tool       string
		autoDetect bool
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid tool provided",
			tool:       "claude",
			autoDetect: false,
			wantErr:    false,
		},
		{
			name:       "empty tool with auto-detect",
			tool:       "",
			autoDetect: true,
			wantErr:    false,
		},
		{
			name:       "empty tool without auto-detect",
			tool:       "",
			autoDetect: false,
			wantErr:    true,
			errMsg:     "--tool flag is required unless --auto-detect is used",
		},
		{
			name:       "tool provided with auto-detect",
			tool:       "gpt",
			autoDetect: true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateTool(tt.tool, tt.autoDetect)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("ValidateTool() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestValidator_ValidateDirectory(t *testing.T) {
	v := NewValidator()

	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "test-validate-dir-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a temp file for testing
	tempFile := filepath.Join(tempDir, "testfile.txt")
	if err := os.WriteFile(tempFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid directory",
			path:    tempDir,
			wantErr: false,
		},
		{
			name:    "empty path allowed",
			path:    "",
			wantErr: false,
		},
		{
			name:    "file instead of directory",
			path:    tempFile,
			wantErr: true,
			errMsg:  "path is not a directory",
		},
		{
			name:    "non-existent path",
			path:    "/non/existent/path/that/should/not/exist",
			wantErr: true,
			errMsg:  "invalid directory",
		},
		{
			name:    "current directory",
			path:    ".",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateDirectory(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDirectory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateDirectory() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestValidator_ValidateFile(t *testing.T) {
	v := NewValidator()

	// Create temp directory and file for testing
	tempDir, err := os.MkdirTemp("", "test-validate-file-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	tempFile := filepath.Join(tempDir, "testfile.txt")
	if err := os.WriteFile(tempFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid file",
			path:    tempFile,
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errMsg:  "file path cannot be empty",
		},
		{
			name:    "directory instead of file",
			path:    tempDir,
			wantErr: true,
			errMsg:  "path is a directory, not a file",
		},
		{
			name:    "non-existent file",
			path:    "/non/existent/file.txt",
			wantErr: true,
			errMsg:  "file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateFile(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateFile() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestValidator_ResolvePath(t *testing.T) {
	v := NewValidator()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{
			name:    "empty path",
			path:    "",
			want:    "",
			wantErr: false,
		},
		{
			name:    "current directory",
			path:    ".",
			want:    cwd,
			wantErr: false,
		},
		{
			name:    "absolute path",
			path:    "/usr/local/bin",
			want:    "/usr/local/bin",
			wantErr: false,
		},
		{
			name:    "relative path",
			path:    "subdir",
			want:    filepath.Join(cwd, "subdir"),
			wantErr: false,
		},
		{
			name:    "relative path with parent",
			path:    "../test",
			want:    filepath.Join(filepath.Dir(cwd), "test"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := v.ResolvePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolvePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolvePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidator_GetDefaultDatabasePath(t *testing.T) {
	v := NewValidator()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	expectedPath := filepath.Join(homeDir, ".ai-memory", "all_conversations.db")

	got, err := v.GetDefaultDatabasePath()
	if err != nil {
		t.Errorf("GetDefaultDatabasePath() error = %v", err)
		return
	}

	if got != expectedPath {
		t.Errorf("GetDefaultDatabasePath() = %v, want %v", got, expectedPath)
	}
}

func TestValidator_GetProjectDatabasePath(t *testing.T) {
	v := NewValidator()

	// Create temp directory for testing
	tempDir, err := os.MkdirTemp("", "test-project-db-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a temp file for testing
	tempFile := filepath.Join(tempDir, "testfile.txt")
	if err := os.WriteFile(tempFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		projectDir string
		want       string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "valid project directory",
			projectDir: tempDir,
			want:       filepath.Join(tempDir, ".ai-memory", "conversations.db"),
			wantErr:    false,
		},
		{
			name:       "file instead of directory",
			projectDir: tempFile,
			want:       "",
			wantErr:    true,
			errMsg:     "invalid project directory",
		},
		{
			name:       "non-existent directory",
			projectDir: "/non/existent/path",
			want:       "",
			wantErr:    true,
			errMsg:     "invalid project directory",
		},
		{
			name:       "relative path",
			projectDir: ".",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := v.GetProjectDatabasePath(tt.projectDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetProjectDatabasePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("GetProjectDatabasePath() error message = %v, want to contain %v", err.Error(), tt.errMsg)
				}
			}
			if !tt.wantErr && tt.want != "" && got != tt.want {
				t.Errorf("GetProjectDatabasePath() = %v, want %v", got, tt.want)
			}
		})
	}
}


// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		   len(s) >= len(substr) && contains(s[1:], substr)
}