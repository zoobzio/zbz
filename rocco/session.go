package rocco

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SessionStore interface for session persistence
type SessionStore interface {
	Create(ctx context.Context, session *Session) error
	Get(ctx context.Context, sessionID string) (*Session, error)
	Update(ctx context.Context, session *Session) error
	Delete(ctx context.Context, sessionID string) error
	DeleteExpired(ctx context.Context) (int, error)
	List(ctx context.Context, identityID string) ([]*Session, error)
}

// MemorySessionStore implements SessionStore with in-memory storage
type MemorySessionStore struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewMemorySessionStore creates a new in-memory session store
func NewMemorySessionStore() *MemorySessionStore {
	store := &MemorySessionStore{
		sessions: make(map[string]*Session),
	}
	
	// Start cleanup goroutine
	go store.cleanupExpired()
	
	return store
}

// Create stores a new session
func (m *MemorySessionStore) Create(ctx context.Context, session *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.sessions[session.ID]; exists {
		return fmt.Errorf("session %s already exists", session.ID)
	}
	
	// Make a copy to avoid reference issues
	sessionCopy := *session
	if sessionCopy.Data == nil {
		sessionCopy.Data = make(map[string]any)
	}
	
	m.sessions[session.ID] = &sessionCopy
	return nil
}

// Get retrieves a session by ID
func (m *MemorySessionStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}
	
	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session %s has expired", sessionID)
	}
	
	// Return a copy
	sessionCopy := *session
	return &sessionCopy, nil
}

// Update updates an existing session
func (m *MemorySessionStore) Update(ctx context.Context, session *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.sessions[session.ID]; !exists {
		return fmt.Errorf("session %s not found", session.ID)
	}
	
	// Make a copy
	sessionCopy := *session
	if sessionCopy.Data == nil {
		sessionCopy.Data = make(map[string]any)
	}
	
	m.sessions[session.ID] = &sessionCopy
	return nil
}

// Delete removes a session
func (m *MemorySessionStore) Delete(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	delete(m.sessions, sessionID)
	return nil
}

// DeleteExpired removes all expired sessions
func (m *MemorySessionStore) DeleteExpired(ctx context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	now := time.Now()
	count := 0
	
	for sessionID, session := range m.sessions {
		if now.After(session.ExpiresAt) {
			delete(m.sessions, sessionID)
			count++
		}
	}
	
	return count, nil
}

// List returns all sessions for an identity
func (m *MemorySessionStore) List(ctx context.Context, identityID string) ([]*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var sessions []*Session
	for _, session := range m.sessions {
		if session.IdentityID == identityID && time.Now().Before(session.ExpiresAt) {
			sessionCopy := *session
			sessions = append(sessions, &sessionCopy)
		}
	}
	
	return sessions, nil
}

// cleanupExpired runs periodically to clean up expired sessions
func (m *MemorySessionStore) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		m.DeleteExpired(context.Background())
	}
}

// RedisSessionStore implements SessionStore using Redis (placeholder)
type RedisSessionStore struct {
	// Redis client would go here
	// client redis.Client
}

// NewRedisSessionStore creates a new Redis-backed session store
func NewRedisSessionStore(addr, password string, db int) *RedisSessionStore {
	// This would initialize Redis client
	return &RedisSessionStore{}
}

// Implement SessionStore interface for Redis
func (r *RedisSessionStore) Create(ctx context.Context, session *Session) error {
	// Implementation would serialize session to JSON and store in Redis
	// with TTL set to session expiry
	return fmt.Errorf("Redis session store not implemented")
}

func (r *RedisSessionStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	return nil, fmt.Errorf("Redis session store not implemented")
}

func (r *RedisSessionStore) Update(ctx context.Context, session *Session) error {
	return fmt.Errorf("Redis session store not implemented")
}

func (r *RedisSessionStore) Delete(ctx context.Context, sessionID string) error {
	return fmt.Errorf("Redis session store not implemented")
}

func (r *RedisSessionStore) DeleteExpired(ctx context.Context) (int, error) {
	return 0, fmt.Errorf("Redis session store not implemented")
}

func (r *RedisSessionStore) List(ctx context.Context, identityID string) ([]*Session, error) {
	return nil, fmt.Errorf("Redis session store not implemented")
}

// DatabaseSessionStore implements SessionStore using SQL database (placeholder)
type DatabaseSessionStore struct {
	// Database connection would go here
	// db *sql.DB
}

// NewDatabaseSessionStore creates a new database-backed session store
func NewDatabaseSessionStore(connectionString string) *DatabaseSessionStore {
	// This would initialize database connection
	return &DatabaseSessionStore{}
}

// Implement SessionStore interface for database
func (d *DatabaseSessionStore) Create(ctx context.Context, session *Session) error {
	// Implementation would insert session into database table
	return fmt.Errorf("Database session store not implemented")
}

func (d *DatabaseSessionStore) Get(ctx context.Context, sessionID string) (*Session, error) {
	return nil, fmt.Errorf("Database session store not implemented")
}

func (d *DatabaseSessionStore) Update(ctx context.Context, session *Session) error {
	return fmt.Errorf("Database session store not implemented")
}

func (d *DatabaseSessionStore) Delete(ctx context.Context, sessionID string) error {
	return fmt.Errorf("Database session store not implemented")
}

func (d *DatabaseSessionStore) DeleteExpired(ctx context.Context) (int, error) {
	return 0, fmt.Errorf("Database session store not implemented")
}

func (d *DatabaseSessionStore) List(ctx context.Context, identityID string) ([]*Session, error) {
	return nil, fmt.Errorf("Database session store not implemented")
}