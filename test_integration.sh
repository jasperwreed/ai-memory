#!/bin/bash

# Integration test script for mem
set -e

echo "=== Mem Integration Test ==="
echo

# Clean up any existing test database
rm -f ~/.ai-memory/test_conversations.db

# Use a test database
export MEM_DB=~/.ai-memory/test_conversations.db

echo "1. Testing capture command..."
./mem capture --db ~/.ai-memory/test_conversations.db --tool claude --project test-project --tags "auth,golang" < test-conversation.txt
echo "✓ Capture successful"
echo

echo "2. Testing list command..."
./mem list --db ~/.ai-memory/test_conversations.db
echo "✓ List successful"
echo

echo "3. Testing search command..."
./mem search --db ~/.ai-memory/test_conversations.db "authentication"
echo "✓ Search successful"
echo

echo "4. Testing stats command..."
./mem stats --db ~/.ai-memory/test_conversations.db
echo "✓ Stats successful"
echo

echo "5. Testing export command..."
./mem export --db ~/.ai-memory/test_conversations.db --id 1 > /tmp/exported_conversation.json
echo "✓ Export successful"
echo "Exported conversation:"
cat /tmp/exported_conversation.json | head -20
echo

echo "6. Creating additional test conversations..."
echo "User: How to handle errors in Go?
Assistant: Use the error interface and return errors from functions." | ./mem capture --db ~/.ai-memory/test_conversations.db --tool aider --project backend

echo "Human: What's the best database for Go?
Assistant: PostgreSQL with pgx driver or SQLite for embedded use cases." | ./mem capture --db ~/.ai-memory/test_conversations.db --tool claude --project database

echo "✓ Additional conversations captured"
echo

echo "7. Testing search with multiple results..."
./mem search --db ~/.ai-memory/test_conversations.db "Go" --limit 5
echo "✓ Multi-result search successful"
echo

echo "8. Testing filtered list..."
./mem list --db ~/.ai-memory/test_conversations.db --tool claude
echo "✓ Filtered list successful"
echo

echo "9. Final stats check..."
./mem stats --db ~/.ai-memory/test_conversations.db
echo

echo "=== All tests passed successfully! ==="
echo
echo "The mem tool is ready to use. You can:"
echo "  - Capture conversations: mem capture --tool <tool> --project <project>"
echo "  - Search conversations: mem search <query>"
echo "  - Browse in TUI: mem browse"
echo "  - View stats: mem stats"
echo
echo "Test database saved at: ~/.ai-memory/test_conversations.db"