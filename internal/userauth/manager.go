package userauth

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"sync"
	"time"
)

type User struct {
	UID      int64
	Username string
}

type Manager struct {
	ttl time.Duration

	mu       sync.Mutex
	sessions map[string]session
	now      func() time.Time
}

type session struct {
	user      User
	expiresAt time.Time
}

func NewManager(ttl time.Duration) *Manager {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &Manager{
		ttl:      ttl,
		sessions: map[string]session{},
		now:      time.Now,
	}
}

func (m *Manager) Login(user User) (string, time.Time, error) {
	if user.UID <= 0 {
		return "", time.Time{}, nil
	}
	token, err := generateToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := m.now().Add(m.ttl)

	m.mu.Lock()
	m.pruneLocked(m.now())
	m.sessions[token] = session{user: user, expiresAt: expiresAt}
	m.mu.Unlock()

	return token, expiresAt, nil
}

func (m *Manager) Validate(token string) (User, time.Time, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return User{}, time.Time{}, false
	}

	now := m.now()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pruneLocked(now)

	session, ok := m.sessions[token]
	if !ok || !session.expiresAt.After(now) {
		delete(m.sessions, token)
		return User{}, time.Time{}, false
	}
	return session.user, session.expiresAt, true
}

func (m *Manager) Revoke(token string) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[token]; !ok {
		return false
	}
	delete(m.sessions, token)
	return true
}

func (m *Manager) pruneLocked(now time.Time) {
	for token, session := range m.sessions {
		if !session.expiresAt.After(now) {
			delete(m.sessions, token)
		}
	}
}

func generateToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}
