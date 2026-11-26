package web

import (
	"crypto/rand"
	"fmt"
	"sync"

	"quiz.com/internal/quiz"
)

// Session associates a quiz and its CSRF token with a unique session ID.
type Session struct {
	ID        string
	Quiz      *quiz.Quiz
	CSRFToken string
}

// QuizStore manages sessions by ID and protects the registry with a mutex.
type QuizStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session // ID -> session
}

func (s *QuizStore) NewSession(questions []quiz.Question) (*Session, error) {
	sessionID, err := newSessionID()
	if err != nil {
		return nil, err
	}

	csrfToken, err := GenerateCSRFToken()
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:        sessionID,
		Quiz:      quiz.New(questions),
		CSRFToken: csrfToken,
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sessions == nil {
		s.sessions = make(map[string]*Session)
	}
	s.sessions[sessionID] = session

	return session, nil
}

func (s *QuizStore) Get(id string) (*Session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[id]
	return session, ok
}

func newSessionID() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", fmt.Errorf("create session ID: %w", err)
	}

	id[6] = (id[6] & 0x0f) | 0x40
	id[8] = (id[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", id[:4], id[4:6], id[6:8], id[8:10], id[10:]), nil
}
