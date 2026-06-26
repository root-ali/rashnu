package identity

import (
	"context"
	"time"

	"github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/store"
	go_cache_store "github.com/eko/gocache/store/go_cache/v4"
	gocache "github.com/patrickmn/go-cache"
)

const TokenTTL = 72 * time.Hour // 3 days — matches JWT expiration

// TokenStore holds at most one active token per user (keyed by email).
// Tokens expire automatically after TokenTTL. Logging in again, changing
// password, or calling Logout replaces/removes the entry immediately.
type TokenStore struct {
	c *cache.Cache[string]
}

func NewTokenStore() *TokenStore {
	client := gocache.New(TokenTTL, 10*time.Minute) // TTL default, cleanup every 10 min
	s := go_cache_store.NewGoCache(client)
	return &TokenStore{c: cache.New[string](s)}
}

func (s *TokenStore) Set(email, token string) {
	_ = s.c.Set(context.Background(), email, token, store.WithExpiration(TokenTTL))
}

func (s *TokenStore) IsValid(email, token string) bool {
	stored, err := s.c.Get(context.Background(), email)
	return err == nil && stored == token
}

func (s *TokenStore) Delete(email string) {
	_ = s.c.Delete(context.Background(), email)
}
