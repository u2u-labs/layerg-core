package server

import (
	"context"
	"sync"
	"time"

	"github.com/gofrs/uuid/v5"
)

type ActiveTokenCache interface {
	Stop()

	// Check if a given user's local session token and refresh token are valid.
	IsValidLocalToken(userID uuid.UUID, exp int64, tokenId string, refreshExp int64, refreshTokenId string) bool
	// Check if a given user's global session token and refresh token are valid.
	IsValidGlobalToken(userID uuid.UUID, exp int64, tokenId string, refreshExp int64, refreshTokenId string) bool
	// Add or update the local and global tokens for a given user.
	Add(userID uuid.UUID, localSessionExp int64, localSessionTokenId string, localRefreshExp int64, localRefreshTokenId string, globalSessionExp int64, globalSessionTokenId string, globalRefreshExp int64, globalRefreshTokenId string)
	// Remove the local and global tokens for a given user.
	Remove(userID uuid.UUID, localSessionTokenId string, globalSessionTokenId string)
	// Remove all tokens (both local and global) for a given user.
	RemoveAll(userID uuid.UUID)

	GetActiveTokens(userID uuid.UUID) (localSessionToken, localRefreshToken, globalSessionToken, globalRefreshToken string, localSessionExp, localRefreshExp, globalSessionExp, globalRefreshExp int64)
}

type activeTokenCacheUser struct {
	localSessionTokens  map[string]int64
	localRefreshTokens  map[string]int64
	globalSessionTokens map[string]int64
	globalRefreshTokens map[string]int64
}

type LocalActiveTokenCache struct {
	sync.RWMutex

	localTokenExpirySec  int64
	globalTokenExpirySec int64

	ctx         context.Context
	ctxCancelFn context.CancelFunc

	cache map[uuid.UUID]*activeTokenCacheUser
}

func NewLocalActiveTokenCache(localTokenExpirySec, globalTokenExpirySec int64) ActiveTokenCache {
	ctx, ctxCancelFn := context.WithCancel(context.Background())

	s := &LocalActiveTokenCache{
		ctx:         ctx,
		ctxCancelFn: ctxCancelFn,

		localTokenExpirySec:  localTokenExpirySec,
		globalTokenExpirySec: globalTokenExpirySec,

		cache: make(map[uuid.UUID]*activeTokenCacheUser),
	}

	go func() {
		ticker := time.NewTicker(2 * time.Duration(localTokenExpirySec) * time.Second)
		for {
			select {
			case <-s.ctx.Done():
				ticker.Stop()
				return
			case t := <-ticker.C:
				ts := t.UTC().Unix()
				s.Lock()
				for userID, cache := range s.cache {
					// Clean up expired local session tokens
					for token, exp := range cache.localSessionTokens {
						if exp <= ts {
							delete(cache.localSessionTokens, token)
						}
					}
					// Clean up expired local refresh tokens
					for token, exp := range cache.localRefreshTokens {
						if exp <= ts {
							delete(cache.localRefreshTokens, token)
						}
					}
					// Clean up expired global session tokens
					for token, exp := range cache.globalSessionTokens {
						if exp <= ts {
							delete(cache.globalSessionTokens, token)
						}
					}
					// Clean up expired global refresh tokens
					for token, exp := range cache.globalRefreshTokens {
						if exp <= ts {
							delete(cache.globalRefreshTokens, token)
						}
					}
					if len(cache.localSessionTokens) == 0 && len(cache.localRefreshTokens) == 0 && len(cache.globalSessionTokens) == 0 && len(cache.globalRefreshTokens) == 0 {
						delete(s.cache, userID)
					}
				}
				s.Unlock()
			}
		}
	}()

	return s
}

func (s *LocalActiveTokenCache) Stop() {
	s.ctxCancelFn()
}

func (s *LocalActiveTokenCache) IsValidLocalToken(userID uuid.UUID, exp int64, tokenId string, refreshExp int64, refreshTokenId string) bool {
	s.RLock()
	cache, found := s.cache[userID]
	if !found {
		s.RUnlock()
		return true
	}
	if exp <= cache.localSessionTokens[tokenId] && refreshExp <= cache.localRefreshTokens[refreshTokenId] {
		s.RUnlock()
		return true
	}
	s.RUnlock()
	return false
}

func (s *LocalActiveTokenCache) IsValidGlobalToken(userID uuid.UUID, exp int64, tokenId string, refreshExp int64, refreshTokenId string) bool {
	s.RLock()
	cache, found := s.cache[userID]
	if !found {
		s.RUnlock()
		return true
	}
	if exp <= cache.globalSessionTokens[tokenId] && refreshExp <= cache.globalRefreshTokens[refreshTokenId] {
		s.RUnlock()
		return true
	}
	s.RUnlock()
	return false
}

func (s *LocalActiveTokenCache) Add(userID uuid.UUID, localSessionExp int64, localSessionTokenId string, localRefreshExp int64, localRefreshTokenId string, globalSessionExp int64, globalSessionTokenId string, globalRefreshExp int64, globalRefreshTokenId string) {
	s.Lock()
	defer s.Unlock()

	cache, found := s.cache[userID]
	if !found {
		cache = &activeTokenCacheUser{
			localSessionTokens:  make(map[string]int64),
			localRefreshTokens:  make(map[string]int64),
			globalSessionTokens: make(map[string]int64),
			globalRefreshTokens: make(map[string]int64),
		}
		s.cache[userID] = cache
	}

	if localSessionTokenId != "" {
		cache.localSessionTokens[localSessionTokenId] = localSessionExp
	}

	if localRefreshTokenId != "" {
		cache.localRefreshTokens[localRefreshTokenId] = localRefreshExp
	}

	if globalSessionTokenId != "" {
		cache.globalSessionTokens[globalSessionTokenId] = globalSessionExp
	}

	if globalRefreshTokenId != "" {
		cache.globalRefreshTokens[globalRefreshTokenId] = globalRefreshExp
	}
}

func (s *LocalActiveTokenCache) GetActiveTokens(userID uuid.UUID) (localSessionToken, localRefreshToken, globalSessionToken, globalRefreshToken string, localSessionExp, localRefreshExp, globalSessionExp, globalRefreshExp int64) {
	s.RLock()
	defer s.RUnlock()

	cache, found := s.cache[userID]
	if !found {
		// No tokens found for this user, return empty values
		return "", "", "", "", 0, 0, 0, 0
	}

	// Get the latest local session and refresh tokens
	for token, exp := range cache.localSessionTokens {
		if exp > localSessionExp {
			localSessionToken = token
			localSessionExp = exp
		}
	}

	for token, exp := range cache.localRefreshTokens {
		if exp > localRefreshExp {
			localRefreshToken = token
			localRefreshExp = exp
		}
	}

	// Get the latest global session and refresh tokens
	for token, exp := range cache.globalSessionTokens {
		if exp > globalSessionExp {
			globalSessionToken = token
			globalSessionExp = exp
		}
	}

	for token, exp := range cache.globalRefreshTokens {
		if exp > globalRefreshExp {
			globalRefreshToken = token
			globalRefreshExp = exp
		}
	}

	return
}

func (s *LocalActiveTokenCache) Remove(userID uuid.UUID, localSessionTokenId string, globalSessionTokenId string) {
	s.Lock()
	cache, found := s.cache[userID]
	if found {
		if localSessionTokenId != "" {
			delete(cache.localSessionTokens, localSessionTokenId)
		}
		if globalSessionTokenId != "" {
			delete(cache.globalSessionTokens, globalSessionTokenId)
		}
	}
	s.Unlock()
}

func (s *LocalActiveTokenCache) RemoveAll(userID uuid.UUID) {
	s.Lock()
	delete(s.cache, userID)
	s.Unlock()
}
