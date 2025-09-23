package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jasperwreed/ai-memory/internal/capture"
	"github.com/jasperwreed/ai-memory/internal/models"
)

type ClaudeScanner struct{}

func NewClaudeScanner() *ClaudeScanner {
	return &ClaudeScanner{}
}

func (s *ClaudeScanner) Name() string {
	return "Claude Code"
}

func (s *ClaudeScanner) ScanPaths() []string {
	home, err := GetHomeDir()
	if err != nil {
		return []string{}
	}

	return []string{
		filepath.Join(home, ".claude", "projects"),
	}
}

func (s *ClaudeScanner) ScanForSessions() ([]SessionInfo, error) {
	var sessions []SessionInfo

	for _, basePath := range s.ScanPaths() {
		if !FileExists(basePath) {
			continue
		}

		projects, err := os.ReadDir(basePath)
		if err != nil {
			continue
		}

		for _, project := range projects {
			if !project.IsDir() {
				continue
			}

			projectPath := filepath.Join(basePath, project.Name())
			sessionFiles, err := os.ReadDir(projectPath)
			if err != nil {
				continue
			}

			for _, file := range sessionFiles {
				if strings.HasSuffix(file.Name(), ".jsonl") {
					fullPath := filepath.Join(projectPath, file.Name())
					info, err := file.Info()
					if err != nil {
						continue
					}

					projectName := extractProjectName(project.Name())

					sessions = append(sessions, SessionInfo{
						Path:        fullPath,
						Tool:        "claude-code",
						ProjectName: projectName,
						Size:        info.Size(),
						ModTime:     info.ModTime().Format("2006-01-02 15:04"),
					})
				}
			}
		}
	}

	return sessions, nil
}

func (s *ClaudeScanner) ParseSession(path string) (*models.Conversation, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open session file: %w", err)
	}
	defer file.Close()

	parser := capture.NewClaudeCodeParserWithPath(path)
	return parser.ParseJSONL(file)
}

func extractProjectName(dirName string) string {
	name := strings.ReplaceAll(dirName, "-", "/")

	if strings.HasPrefix(name, "/") {
		parts := strings.Split(name, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	return dirName
}