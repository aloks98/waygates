package service

import (
	"testing"
	"time"

	"github.com/aloks98/goauth/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOneTimeCodeStore_IssueConsume(t *testing.T) {
	s := NewOneTimeCodeStore(time.Minute)
	pair := &token.Pair{AccessToken: "a", RefreshToken: "r"}

	code, err := s.Issue(pair)
	require.NoError(t, err)
	require.NotEmpty(t, code)

	got, ok := s.Consume(code)
	require.True(t, ok)
	assert.Equal(t, "a", got.AccessToken)
}

func TestOneTimeCodeStore_SingleUse(t *testing.T) {
	s := NewOneTimeCodeStore(time.Minute)
	code, _ := s.Issue(&token.Pair{AccessToken: "a"})
	_, ok1 := s.Consume(code)
	_, ok2 := s.Consume(code)
	assert.True(t, ok1)
	assert.False(t, ok2, "a code must be consumable only once")
}

func TestOneTimeCodeStore_Expiry(t *testing.T) {
	s := NewOneTimeCodeStore(time.Minute)
	base := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return base }
	code, _ := s.Issue(&token.Pair{AccessToken: "a"})

	s.now = func() time.Time { return base.Add(2 * time.Minute) } // past TTL
	_, ok := s.Consume(code)
	assert.False(t, ok, "an expired code must not be consumable")
}

func TestOneTimeCodeStore_UnknownCode(t *testing.T) {
	s := NewOneTimeCodeStore(time.Minute)
	_, ok := s.Consume("nope")
	assert.False(t, ok)
}
