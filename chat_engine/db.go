package chat_engine

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type DB struct {
	db *sql.DB
}

func NewDB(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	database := &DB{db: db}

	// Initialize schema
	if err := database.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return database, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) initSchema() error {
	// Create conversations table
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create conversations table: %w", err)
	}

	// Create messages table
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			tool_call_id TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create messages table: %w", err)
	}

	// Create tool_calls table
	_, err = d.db.Exec(`
		CREATE TABLE IF NOT EXISTS tool_calls (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			message_id TEXT NOT NULL,
			tool_call_id TEXT NOT NULL,
			type TEXT NOT NULL,
			name TEXT NOT NULL,
			arguments TEXT NOT NULL,
			FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create tool_calls table: %w", err)
	}

	// Create indexes for better query performance
	_, err = d.db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
		CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);
		CREATE INDEX IF NOT EXISTS idx_tool_calls_message_id ON tool_calls(message_id);
	`)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// SaveConversation creates or updates a conversation
func (d *DB) SaveConversation(conv *Conversation) error {
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert or update conversation
	_, err = tx.Exec(`
		INSERT INTO conversations (id, updated_at)
		VALUES (?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET updated_at = CURRENT_TIMESTAMP
	`, conv.ID)
	if err != nil {
		return fmt.Errorf("failed to save conversation: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// SaveMessage saves a message to the database
func (d *DB) SaveMessage(conversationID string, msg *Message) error {
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Ensure conversation exists
	_, err = tx.Exec(`
		INSERT INTO conversations (id, updated_at)
		VALUES (?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET updated_at = CURRENT_TIMESTAMP
	`, conversationID)
	if err != nil {
		return fmt.Errorf("failed to ensure conversation exists: %w", err)
	}

	// Insert message
	_, err = tx.Exec(`
		INSERT INTO messages (id, conversation_id, role, content, tool_call_id)
		VALUES (?, ?, ?, ?, ?)
	`, msg.ID, conversationID, msg.Role, msg.Content, msg.TollCallID)
	if err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	// Insert tool calls if any
	for _, toolCall := range msg.ToolCalls {
		_, err = tx.Exec(`
			INSERT INTO tool_calls (message_id, tool_call_id, type, name, arguments)
			VALUES (?, ?, ?, ?, ?)
		`, msg.ID, toolCall.ID, toolCall.Type, toolCall.Name, toolCall.Arguments)
		if err != nil {
			return fmt.Errorf("failed to insert tool call: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// LoadConversation loads a conversation with all its messages from the database
func (d *DB) LoadConversation(conversationID string) (*Conversation, error) {
	// Check if conversation exists
	var exists bool
	err := d.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM conversations WHERE id = ?)
	`, conversationID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check conversation existence: %w", err)
	}

	if !exists {
		return nil, nil
	}

	// Load messages
	rows, err := d.db.Query(`
		SELECT id, role, content, tool_call_id
		FROM messages
		WHERE conversation_id = ?
		ORDER BY created_at ASC
	`, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	messages := make([]*Message, 0)
	messageMap := make(map[string]*Message)

	for rows.Next() {
		var msg Message
		var toolCallID string
		err := rows.Scan(&msg.ID, &msg.Role, &msg.Content, &toolCallID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		msg.TollCallID = toolCallID
		msg.ToolCalls = make([]ToolCall, 0)

		messages = append(messages, &msg)
		messageMap[msg.ID] = &msg
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating messages: %w", err)
	}

	// Load tool calls for all messages
	if len(messages) > 0 {
		messageIDs := make([]interface{}, len(messages))
		for i, msg := range messages {
			messageIDs[i] = msg.ID
		}

		// Build query with placeholders
		placeholders := ""
		for i := range messageIDs {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
		}

		query := fmt.Sprintf(`
			SELECT message_id, tool_call_id, type, name, arguments
			FROM tool_calls
			WHERE message_id IN (%s)
			ORDER BY id ASC
		`, placeholders)

		toolRows, err := d.db.Query(query, messageIDs...)
		if err != nil {
			return nil, fmt.Errorf("failed to query tool calls: %w", err)
		}
		defer toolRows.Close()

		for toolRows.Next() {
			var messageID string
			var toolCall ToolCall
			err := toolRows.Scan(&messageID, &toolCall.ID, &toolCall.Type, &toolCall.Name, &toolCall.Arguments)
			if err != nil {
				return nil, fmt.Errorf("failed to scan tool call: %w", err)
			}

			if msg, ok := messageMap[messageID]; ok {
				msg.ToolCalls = append(msg.ToolCalls, toolCall)
			}
		}

		if err := toolRows.Err(); err != nil {
			return nil, fmt.Errorf("error iterating tool calls: %w", err)
		}
	}

	conv := &Conversation{
		ID:       conversationID,
		Messages: messages,
	}

	return conv, nil
}

// ListConversations returns all conversation IDs
func (d *DB) ListConversations() ([]string, error) {
	rows, err := d.db.Query(`
		SELECT id
		FROM conversations
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query conversations: %w", err)
	}
	defer rows.Close()

	conversationIDs := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan conversation ID: %w", err)
		}
		conversationIDs = append(conversationIDs, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating conversations: %w", err)
	}

	return conversationIDs, nil
}

// DeleteConversation deletes a conversation and all its messages
func (d *DB) DeleteConversation(conversationID string) error {
	_, err := d.db.Exec(`DELETE FROM conversations WHERE id = ?`, conversationID)
	if err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}
	return nil
}

