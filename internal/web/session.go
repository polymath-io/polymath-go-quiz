package web

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"quiz.com/internal/quiz"
)

const sessionTTL = time.Hour

// Session associates a quiz and its CSRF token with a unique session ID.
type Session struct {
	ID        string
	Quiz      *quiz.Quiz
	CSRFToken string
	ExpiresAt time.Time
}

// QuizStore manages sessions by ID and protects the registry with a mutex.
type QuizStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session // ID -> session
}

// NewSession creates a session for a new quiz over questions, which must pass
// quiz.ValidateQuestions.
func (s *QuizStore) NewSession(questions []quiz.Question) (*Session, error) {
	if err := quiz.ValidateQuestions(questions); err != nil {
		return nil, err
	}

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
		ExpiresAt: time.Now().Add(sessionTTL),
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
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[id]
	if ok && !time.Now().Before(session.ExpiresAt) {
		delete(s.sessions, id)

		return nil, false
	}

	return session, ok
}

// CleanupExpired removes sessions whose expiry time has passed and returns the
// number of sessions removed.
func (s *QuizStore) CleanupExpired(now time.Time) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	removed := 0
	for id, session := range s.sessions {
		if !now.Before(session.ExpiresAt) {
			delete(s.sessions, id)
			removed++
		}
	}

	return removed
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
