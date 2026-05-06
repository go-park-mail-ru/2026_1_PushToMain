package utils

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJWTManager(t *testing.T) {
	secret := "my-secret"
	expire := 24
	jm := NewJWTManager(secret, expire)

	assert.Equal(t, secret, jm.Secret)
	assert.Equal(t, expire, jm.Expire)
}

func TestJWTManager_TTL(t *testing.T) {
	jm := NewJWTManager("secret", 5)
	assert.Equal(t, 5*time.Hour, jm.TTL())
}

func TestJWTManager_GenerateJWT(t *testing.T) {
	jm := NewJWTManager("test-secret", 1)

	t.Run("success", func(t *testing.T) {
		token, err := jm.GenerateJWT(123)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		parts := splitToken(t, token)
		assert.Len(t, parts, 3)

		headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
		require.NoError(t, err)
		var header jwtHeader
		err = json.Unmarshal(headerJSON, &header)
		require.NoError(t, err)
		assert.Equal(t, "HS256", header.Alg)
		assert.Equal(t, "JWT", header.Typ)

		payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
		require.NoError(t, err)
		var payload JwtPayload
		err = json.Unmarshal(payloadJSON, &payload)
		require.NoError(t, err)
		assert.Equal(t, int64(123), payload.UserId)
		expectedExp := time.Now().Add(jm.TTL()).Unix()
		assert.InDelta(t, expectedExp, payload.Exp, 2) // allow 2 seconds delta

		unsigned := parts[0] + "." + parts[1]
		expectedSig := jm.sign(unsigned)
		assert.Equal(t, expectedSig, parts[2])
	})
}

func TestJWTManager_ValidateJWT(t *testing.T) {
	jm := NewJWTManager("test-secret", 1)

	t.Run("valid token", func(t *testing.T) {
		token, err := jm.GenerateJWT(456)
		require.NoError(t, err)

		payload, err := jm.ValidateJWT(token)
		require.NoError(t, err)
		assert.Equal(t, int64(456), payload.UserId)
		assert.True(t, payload.Exp > time.Now().Unix())
	})

	t.Run("invalid token format - wrong parts", func(t *testing.T) {
		_, err := jm.ValidateJWT("invalid.token")
		assert.EqualError(t, err, "invalid token format")
	})

	t.Run("invalid token format - empty string", func(t *testing.T) {
		_, err := jm.ValidateJWT("")
		assert.EqualError(t, err, "invalid token format")
	})

	t.Run("invalid signature", func(t *testing.T) {
		token, err := jm.GenerateJWT(789)
		require.NoError(t, err)
		parts := splitToken(t, token)

		tamperedSig := parts[2] + "x"
		invalidToken := parts[0] + "." + parts[1] + "." + tamperedSig

		_, err = jm.ValidateJWT(invalidToken)
		assert.EqualError(t, err, "invalid signature")
	})

	t.Run("invalid signature - wrong secret", func(t *testing.T) {
		jm2 := NewJWTManager("different-secret", 1)
		token, err := jm2.GenerateJWT(999)
		require.NoError(t, err)

		_, err = jm.ValidateJWT(token)
		assert.EqualError(t, err, "invalid signature")
	})

	t.Run("expired token", func(t *testing.T) {
		payload := JwtPayload{
			UserId: 111,
			Exp:    time.Now().Add(-1 * time.Hour).Unix(),
		}
		payloadJSON, _ := json.Marshal(payload)
		header := jwtHeader{Alg: "HS256", Typ: "JWT"}
		headerJSON, _ := json.Marshal(header)

		unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(payloadJSON)
		sig := jm.sign(unsigned)
		token := unsigned + "." + sig

		_, err := jm.ValidateJWT(token)
		assert.EqualError(t, err, "token expired")
	})

	t.Run("malformed payload base64", func(t *testing.T) {
		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
		invalidPayload := "not-valid-base64!@#"
		unsigned := header + "." + invalidPayload
		sig := jm.sign(unsigned)
		token := unsigned + "." + sig

		_, err := jm.ValidateJWT(token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "illegal base64 data")
	})

	t.Run("malformed payload JSON", func(t *testing.T) {
		header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
		invalidPayloadJSON := base64.RawURLEncoding.EncodeToString([]byte(`{"user_id": "not an int"}`))
		unsigned := header + "." + invalidPayloadJSON
		sig := jm.sign(unsigned)
		token := unsigned + "." + sig

		_, err := jm.ValidateJWT(token)
		assert.Error(t, err)
	})
}

func TestJWTManager_ValidateJWT_EdgeCases(t *testing.T) {
	jm := NewJWTManager("secret", 1)

	t.Run("token with extra dots", func(t *testing.T) {
		_, err := jm.ValidateJWT("a.b.c.d")
		assert.EqualError(t, err, "invalid token format")
	})

	t.Run("token with spaces", func(t *testing.T) {
		token, _ := jm.GenerateJWT(1)
		_, err := jm.ValidateJWT(" " + token + " ")
		assert.EqualError(t, err, "invalid signature")
	})
}

func splitToken(t *testing.T, token string) []string {
	t.Helper()
	parts := make([]string, 0)
	last := 0
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			parts = append(parts, token[last:i])
			last = i + 1
		}
	}
	parts = append(parts, token[last:])
	return parts
}
