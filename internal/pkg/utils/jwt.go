package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type JwtPayload struct {
	UserId int64 `json:"user_id"`
	Exp    int64 `json:"exp"`
}

type JWTManager struct {
	Secret string `mapstructure:"secret"`
	Expire int    `mapstructure:"expire_hours"`
}

func NewJWTManager(secret string, expire int) *JWTManager {
	return &JWTManager{
		Secret: secret,
		Expire: expire,
	}
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

func (j *JWTManager) TTL() time.Duration {
	return time.Duration(j.Expire) * time.Hour
}

func (j *JWTManager) GenerateJWT(userId int64) (string, error) {
	header := jwtHeader{
		Alg: "HS256",
		Typ: "JWT",
	}
	payload := JwtPayload{
		UserId: userId,
		Exp:    time.Now().Add(time.Duration(j.Expire) * time.Hour).Unix(),
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	headerEncoded := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadJSON)

	unsignedToken := headerEncoded + "." + payloadEncoded

	signature := j.sign(unsignedToken)

	return unsignedToken + "." + signature, nil
}

func (j *JWTManager) sign(data string) string {
	h := hmac.New(sha256.New, []byte(j.Secret))
	h.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func (j *JWTManager) ValidateJWT(token string) (*JwtPayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	unsigned := parts[0] + "." + parts[1]
	signature := j.sign(unsigned)

	if !hmac.Equal([]byte(signature), []byte(parts[2])) {
		return nil, errors.New("invalid signature")
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	var payload JwtPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return nil, err
	}

	if time.Now().Unix() > payload.Exp {
		return nil, errors.New("token expired")
	}

	return &payload, nil
}
