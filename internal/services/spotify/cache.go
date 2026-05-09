package spotify

import (
	"jota/server/internal/kv"
	"time"
)

// Una sola entrada: el token web no depende de instancias en memoria para poder reutilizarse entre procesos.
const spotifyTokenCacheKey = "spotify-web-access-token"

var tokenBucket = kv.UseBucket("spotify-token")

// accessTokenValid matches the rules used for the KV cache: non-empty bearer and,
// when Spotify sends an expiry, at least ~30s of remaining lifetime.
func accessTokenValid(t *userToken) bool {
	if t == nil || t.AccessToken == "" {
		return false
	}
	if t.AccessTokenExpirationTimestampMs > 0 {
		exp := time.UnixMilli(t.AccessTokenExpirationTimestampMs)
		if !exp.After(time.Now().Add(30 * time.Second)) {
			return false
		}
	}
	return true
}

func loadTokenFromCache() (*userToken, bool) {
	if err := kv.EnsureStarted(); err != nil {
		return nil, false
	}

	var token userToken
	if err := tokenBucket.GetObject(spotifyTokenCacheKey, &token); err != nil {
		return nil, false
	}
	if !accessTokenValid(&token) {
		return nil, false
	}
	return &token, true
}

func clearTokenCache() error {
	if err := kv.EnsureStarted(); err != nil {
		return err
	}
	return tokenBucket.Delete(spotifyTokenCacheKey)
}

func saveTokenToCache(token userToken) error {
	if err := kv.EnsureStarted(); err != nil {
		return err
	}
	var ttl time.Duration
	if token.AccessTokenExpirationTimestampMs > 0 {
		ttl = time.Until(time.UnixMilli(token.AccessTokenExpirationTimestampMs))
	}
	if ttl <= 0 {
		ttl = 55 * time.Minute
	}
	if ttl < time.Minute {
		ttl = time.Minute
	}
	return tokenBucket.SetObject(spotifyTokenCacheKey, token, ttl)
}
