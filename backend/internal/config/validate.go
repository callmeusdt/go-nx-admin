package config

import (
	"errors"
	"fmt"
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
	return nil
}

// Warnings lists settings that are accepted but have no effect, so a stale
// config file does not silently mislead the operator.
func (c *Config) Warnings() []string {
	var w []string
	if c.CORS.Origins != "" {
		w = append(w, "cors.origins is ignored: the application sends no Access-Control-* headers; configure CORS in the reverse proxy")
	}
	return w
}
