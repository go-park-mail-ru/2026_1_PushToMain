package tools

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var secretKey = []byte("super-secret-key")

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type jwtPayload struct {
	Email string `json:"email"`
	Exp   int64  `json:"exp"`
}

func GenerateJWT(email, name, surname string) (string, error) {
	header := jwtHeader{
		Alg: "HS256",
		Typ: "JWT",
	}

	payload := jwtPayload{
		Email: email,
		Exp:   time.Now().Add(24 * time.Hour).Unix(),
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

	signature := sign(unsignedToken)

	return unsignedToken + "." + signature, nil
}

func sign(data string) string {
	h := hmac.New(sha256.New, secretKey)
	h.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func ValidateJWT(token string) (*jwtPayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	unsigned := parts[0] + "." + parts[1]
	signature := sign(unsigned)

	if !hmac.Equal([]byte(signature), []byte(parts[2])) {
		return nil, errors.New("invalid signature")
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	var payload jwtPayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return nil, err
	}

	if time.Now().Unix() > payload.Exp {
		return nil, errors.New("token expired")
	}

	return &payload, nil
}
