package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// MinJWTSecretLen is the shortest accepted HMAC signing key.
const MinJWTSecretLen = 32

// knownWeakSecrets are values that shipped in earlier templates and are public.
var knownWeakSecrets = map[string]bool{
	"nx-admin-jwt-secret-2024": true,
}

// Validate rejects settings that would leave a deployment insecure or unable
// to start. Both app.Run and app.NewRuntime call it, so the two entry points
// enforce the same floor.
func (c *Config) Validate() error {
	if c.Server.Host == "" || c.Server.Port < 1 || c.Server.Port > 65535 {
		return errors.New("config: explicit server host and a valid port are required")
	}
	if len(c.JWT.Secret) < MinJWTSecretLen || knownWeakSecrets[c.JWT.Secret] {
		return fmt.Errorf("config: jwt.secret must be at least %d bytes and not a published default (leave it unset to use a generated key)", MinJWTSecretLen)
	}
	return c.ValidateCORS()
}

// ValidateCORS accepts an empty value (no CORS headers are sent, which is what
// a deployment fronted by a reverse proxy that owns CORS wants). A non-empty
// value must be exact http(s) origins; wildcards would let any site call the
// API with a stolen token.
func (c *Config) ValidateCORS() error {
	if c.CORS.Origins == "" {
		return nil
	}
	if strings.Contains(c.CORS.Origins, "*") {
		return errors.New("config: cors.origins must list exact origins (no wildcard); leave it empty to send no CORS headers")
	}
	for _, origin := range strings.Split(c.CORS.Origins, ",") {
		u, err := url.Parse(strings.TrimSpace(origin))
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return errors.New("config: invalid cors origin")
		}
	}
	return nil
}
