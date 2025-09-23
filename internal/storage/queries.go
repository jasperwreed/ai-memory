package storage

// Database schema queries
const (
	queryCreateProjectsTable = `CREATE TABLE IF NOT EXISTS projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_path TEXT UNIQUE NOT NULL
	)`

	queryCreateConversationsTable = `CREATE TABLE IF NOT EXISTS conversations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		tool TEXT NOT NULL,
		project TEXT,
		project_id INTEGER,
		tags TEXT,
		session_id TEXT,
		source_path TEXT,
		audit_shard TEXT,
		raw_json TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	)`

	queryCreateMessagesTable = `CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		conversation_id INTEGER NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		token_count INTEGER DEFAULT 0,
		FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
	)`

	queryCreateMessagesFTS = `CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
		content,
		content=messages,
		content_rowid=id
	)`

	queryCreateIndexMessagesConversation = `CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id)`
	queryCreateIndexConversationsTool    = `CREATE INDEX IF NOT EXISTS idx_conversations_tool ON conversations(tool)`
	queryCreateIndexConversationsProject = `CREATE INDEX IF NOT EXISTS idx_conversations_project ON conversations(project)`
	queryCreateIndexConversationsProjectID = `CREATE INDEX IF NOT EXISTS idx_conversations_project_id ON conversations(project_id)`
	queryCreateIndexConversationsCreated = `CREATE INDEX IF NOT EXISTS idx_conversations_created ON conversations(created_at)`
	queryCreateIndexConversationsSession  = `CREATE INDEX IF NOT EXISTS idx_conversations_session ON conversations(session_id)`
	queryCreateIndexConversationsSource  = `CREATE INDEX IF NOT EXISTS idx_conversations_source ON conversations(source_path)`

	queryCreateMessagesInsertTrigger = `CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages
	BEGIN
		INSERT INTO messages_fts(rowid, content) VALUES (new.id, new.content);
	END`

	queryCreateMessagesDeleteTrigger = `CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages
	BEGIN
		DELETE FROM messages_fts WHERE rowid = old.id;
	END`

	queryCreateMessagesUpdateTrigger = `CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages
	BEGIN
		UPDATE messages_fts SET content = new.content WHERE rowid = new.id;
	END`

	queryInsertProject = `INSERT OR IGNORE INTO projects (project_path) VALUES (?)`

	querySelectProjectID = `SELECT id FROM projects WHERE project_path = ?`

	queryInsertConversation = `INSERT INTO conversations (title, tool, project, project_id, tags, session_id, source_path, audit_shard, raw_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	queryInsertMessage = `INSERT INTO messages (conversation_id, role, content, timestamp, token_count)
		VALUES (?, ?, ?, ?, ?)`

	querySelectConversation = `SELECT id, title, tool, project, project_id, tags, session_id, source_path, audit_shard, raw_json, created_at, updated_at
		FROM conversations WHERE id = ?`

	querySelectMessages = `SELECT id, conversation_id, role, content, timestamp, token_count
		FROM messages WHERE conversation_id = ? ORDER BY timestamp`

	queryDeleteConversation = `DELETE FROM conversations WHERE id = ?`

	querySearchConversations = `
		SELECT DISTINCT
			c.id, c.title, c.tool, c.project, c.tags, c.created_at, c.updated_at,
			m.content, bm25(messages_fts) as score
		FROM messages_fts
		JOIN messages m ON messages_fts.rowid = m.id
		JOIN conversations c ON m.conversation_id = c.id
		WHERE messages_fts MATCH ?
		ORDER BY score DESC
		LIMIT ?`

	queryCountConversations = `SELECT COUNT(*) FROM conversations`
	queryCountMessages      = `SELECT COUNT(*) FROM messages`
	querySumTokens          = `SELECT COALESCE(SUM(token_count), 0) FROM messages`
	queryGroupByTool        = `SELECT tool, COUNT(*) FROM conversations GROUP BY tool`
	queryGroupByProject     = `SELECT project, COUNT(*) FROM conversations WHERE project IS NOT NULL GROUP BY project`
)