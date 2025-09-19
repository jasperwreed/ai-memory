#!/bin/bash

# Mem - AI Conversation Memory Installation Script
# Builds and installs mem CLI tool

set -e

echo "Building Mem..."
go build -tags "sqlite_fts5" -o mem cmd/mem/main.go

echo "Installing to /usr/local/bin..."
sudo cp mem /usr/local/bin/mem

echo ""
echo "✓ Mem installed successfully!"
echo ""
echo "Usage: mem [command]"
echo ""
echo "Examples:"
echo "  mem capture --tool claude         # Capture from stdin"
echo "  mem search \"authentication\"       # Search conversations"
echo "  mem list                          # List recent conversations"
echo "  mem browse                        # Open TUI browser"
echo "  mem import --claude-project       # Import Claude Code sessions"
echo ""
echo "Run 'mem --help' for more information"