package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jasperwreed/ai-memory/internal/models"
)

type Scanner interface {
	Name() string
	ScanPaths() []string
	ScanForSessions() ([]SessionInfo, error)
	ParseSession(path string) (*models.Conversation, error)
}

type SessionInfo struct {
	Path        string
	Tool        string
	ProjectName string
	Size        int64
	ModTime     string
}

type ScanResult struct {
	Tool         string
	SessionsFound int
	Imported     int
	Failed       int
	Errors       []string
}

func GetHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return home, nil
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func FindFiles(root string, pattern string) ([]string, error) {
	var files []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if !info.IsDir() && matchesPattern(path, pattern) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func matchesPattern(path, pattern string) bool {
	matched, _ := filepath.Match(pattern, filepath.Base(path))
	return matched
}

func GetProjectFromPath(path string) string {
	dir := filepath.Dir(path)
	parts := strings.Split(dir, string(filepath.Separator))

	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" && !strings.HasPrefix(parts[i], ".") {
			return parts[i]
		}
	}

	return "unknown"
}