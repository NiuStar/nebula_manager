package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// AuthService manages simple username/password verification and signed session tokens.
type AuthService struct {
	username      string
	password      string
	secret        []byte
	sessionTTL    time.Duration
	cookieName    string
	secureCookies bool
	staticToken   string
}

// NewAuthService constructs an AuthService.
func NewAuthService(username, password, secret string, secureCookie bool, staticToken string) *AuthService {
	ttl := 24 * time.Hour
	if len(secret) == 0 {
		secret = "nebula-session-secret"
	}
	return &AuthService{
		username:      username,
		password:      password,
		secret:        []byte(secret),
		sessionTTL:    ttl,
		cookieName:    "nebula_session",
		secureCookies: secureCookie,
		staticToken:   staticToken,
	}
}

// CookieName returns the configured session cookie name.
func (s *AuthService) CookieName() string {
	return s.cookieName
}

// SecureCookies indicates whether session cookies should be marked Secure.
func (s *AuthService) SecureCookies() bool {
	return s.secureCookies
}

// StaticToken returns the configured static access token, if any.
func (s *AuthService) StaticToken() string {
	return s.staticToken
}

// AdminUsername exposes the configured admin username.
func (s *AuthService) AdminUsername() string {
	return s.username
}

// ValidateCredentials checks provided credentials against the configured admin account.
func (s *AuthService) ValidateCredentials(username, password string) bool {
	return subtleConstantCompare(username, s.username) && subtleConstantCompare(password, s.password)
}

// IssueToken creates a signed session token for the specified username.
func (s *AuthService) IssueToken(username string) (token string, expiresAt time.Time, err error) {
	expiresAt = time.Now().Add(s.sessionTTL)
	nonce := make([]byte, 16)
	if _, err = rand.Read(nonce); err != nil {
		return "", time.Time{}, fmt.Errorf("generate nonce: %w", err)
	}

	payload := fmt.Sprintf("%s|%d|%s", username, expiresAt.Unix(), hex.EncodeToString(nonce))
	signature := signPayload(payload, s.secret)

	token = base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(signature)
	return token, expiresAt, nil
}

// ValidateToken parses and validates a signed session token.
func (s *AuthService) ValidateToken(token string) (username string, expiresAt time.Time, err error) {
	if token == "" {
		return "", time.Time{}, errors.New("empty token")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", time.Time{}, errors.New("token format invalid")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", time.Time{}, errors.New("token payload decode failed")
	}

	providedSig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", time.Time{}, errors.New("token signature decode failed")
	}

	expectedSig := signPayload(string(payloadBytes), s.secret)
	if !hmac.Equal(providedSig, expectedSig) {
		return "", time.Time{}, errors.New("token signature mismatch")
	}

	segments := strings.Split(string(payloadBytes), "|")
	if len(segments) < 3 {
		return "", time.Time{}, errors.New("token payload invalid")
	}

	username = segments[0]
	unixTs, err := strconv.ParseInt(segments[1], 10, 64)
	if err != nil {
		return "", time.Time{}, errors.New("token expiry invalid")
	}
	expiresAt = time.Unix(unixTs, 0)
	if time.Now().After(expiresAt) {
		return "", time.Time{}, errors.New("token expired")
	}

	return username, expiresAt, nil
}

func signPayload(payload string, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	return mac.Sum(nil)
}

func subtleConstantCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
