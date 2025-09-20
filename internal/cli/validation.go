package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// Validator provides methods for validating CLI inputs
type Validator struct{}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateTool checks if a tool name is provided when required
func (v *Validator) ValidateTool(tool string, autoDetect bool) error {
	if tool == "" && !autoDetect {
		return fmt.Errorf("--tool flag is required unless --auto-detect is used")
	}
	return nil
}

// ValidateDirectory checks if a directory path is valid
func (v *Validator) ValidateDirectory(path string) error {
	if path == "" {
		return nil // Empty path is allowed, will use default
	}

	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("invalid directory: %w", err)
	}

	if !stat.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	return nil
}

// ValidateFile checks if a file path is valid and exists
func (v *Validator) ValidateFile(path string) error {
	if path == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	stat, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	if stat.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", path)
	}

	return nil
}

// ResolvePath resolves a path to an absolute path
func (v *Validator) ResolvePath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	if path == "." {
		return os.Getwd()
	}

	if filepath.IsAbs(path) {
		return path, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	return filepath.Join(cwd, path), nil
}

// GetDefaultDatabasePath returns the default database path
func (v *Validator) GetDefaultDatabasePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".ai-memory", "all_conversations.db"), nil
}

// GetProjectDatabasePath returns the database path for a specific project
func (v *Validator) GetProjectDatabasePath(projectDir string) (string, error) {
	resolvedDir, err := v.ResolvePath(projectDir)
	if err != nil {
		return "", err
	}

	if err := v.ValidateDirectory(resolvedDir); err != nil {
		return "", fmt.Errorf("invalid project directory: %w", err)
	}

	return filepath.Join(resolvedDir, ".ai-memory", "conversations.db"), nil
}