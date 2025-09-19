#!/bin/bash

# Integration test script for ai-memory
set -e

echo "=== AI Memory Integration Test ==="
echo

# Clean up any existing test database
rm -f ~/.ai-memory/test_conversations.db

# Use a test database
export AI_MEMORY_DB=~/.ai-memory/test_conversations.db

echo "1. Testing capture command..."
./ai-memory capture --db ~/.ai-memory/test_conversations.db --tool claude --project test-project --tags "auth,golang" < test-conversation.txt
echo "✓ Capture successful"
echo

echo "2. Testing list command..."
./ai-memory list --db ~/.ai-memory/test_conversations.db
echo "✓ List successful"
echo

echo "3. Testing search command..."
./ai-memory search --db ~/.ai-memory/test_conversations.db "authentication"
echo "✓ Search successful"
echo

echo "4. Testing stats command..."
./ai-memory stats --db ~/.ai-memory/test_conversations.db
echo "✓ Stats successful"
echo

echo "5. Testing export command..."
./ai-memory export --db ~/.ai-memory/test_conversations.db --id 1 > /tmp/exported_conversation.json
echo "✓ Export successful"
echo "Exported conversation:"
cat /tmp/exported_conversation.json | head -20
echo

echo "6. Creating additional test conversations..."
echo "User: How to handle errors in Go?
Assistant: Use the error interface and return errors from functions." | ./ai-memory capture --db ~/.ai-memory/test_conversations.db --tool aider --project backend

echo "Human: What's the best database for Go?
Assistant: PostgreSQL with pgx driver or SQLite for embedded use cases." | ./ai-memory capture --db ~/.ai-memory/test_conversations.db --tool claude --project database

echo "✓ Additional conversations captured"
echo

echo "7. Testing search with multiple results..."
./ai-memory search --db ~/.ai-memory/test_conversations.db "Go" --limit 5
echo "✓ Multi-result search successful"
echo

echo "8. Testing filtered list..."
./ai-memory list --db ~/.ai-memory/test_conversations.db --tool claude
echo "✓ Filtered list successful"
echo

echo "9. Final stats check..."
./ai-memory stats --db ~/.ai-memory/test_conversations.db
echo

echo "=== All tests passed successfully! ==="
echo
echo "The ai-memory tool is ready to use. You can:"
echo "  - Capture conversations: ai-memory capture --tool <tool> --project <project>"
echo "  - Search conversations: ai-memory search <query>"
echo "  - Browse in TUI: ai-memory browse"
echo "  - View stats: ai-memory stats"
echo
echo "Test database saved at: ~/.ai-memory/test_conversations.db"