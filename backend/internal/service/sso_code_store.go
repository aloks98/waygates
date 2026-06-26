package service

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"

	"github.com/aloks98/goauth/token"
)

type codeEntry struct {
	pair      *token.Pair
	expiresAt time.Time
}

// OneTimeCodeStore holds freshly minted JWT pairs under opaque single-use codes
// with a short TTL, so the SSO callback can hand tokens to the SPA without
// putting them in the URL. Safe for concurrent use; single-instance only.
type OneTimeCodeStore struct {
	mu      sync.Mutex
	entries map[string]codeEntry
	ttl     time.Duration
	now     func() time.Time
}

// NewOneTimeCodeStore creates a store whose codes expire after ttl.
func NewOneTimeCodeStore(ttl time.Duration) *OneTimeCodeStore {
	return &OneTimeCodeStore{
		entries: make(map[string]codeEntry),
		ttl:     ttl,
		now:     time.Now,
	}
}

// Issue stores pair under a new random code and returns the code.
func (s *OneTimeCodeStore) Issue(pair *token.Pair) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := base64.RawURLEncoding.EncodeToString(b)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.pruneLocked()
	s.entries[code] = codeEntry{pair: pair, expiresAt: s.now().Add(s.ttl)}
	return code, nil
}

// Consume returns the pair for code and deletes it; ok is false if the code is
// unknown or expired.
func (s *OneTimeCodeStore) Consume(code string) (*token.Pair, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[code]
	if !ok {
		return nil, false
	}
	delete(s.entries, code)
	if s.now().After(e.expiresAt) {
		return nil, false
	}
	return e.pair, true
}

// pruneLocked drops expired entries; caller holds the lock.
func (s *OneTimeCodeStore) pruneLocked() {
	now := s.now()
	for k, e := range s.entries {
		if now.After(e.expiresAt) {
			delete(s.entries, k)
		}
	}
}
