package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	secret             []byte
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration

	// In-memory token store for refresh token rotation and logout
	mu            sync.RWMutex
	refreshTokens map[string]string // refreshToken -> username
	blacklist     map[string]bool   // blacklisted access tokens
}

func NewJWTService(secret string, accessExpiry, refreshExpiry time.Duration) *JWTService {
	return &JWTService{
		secret:             []byte(secret),
		accessTokenExpiry:  accessExpiry,
		refreshTokenExpiry: refreshExpiry,
		refreshTokens:      make(map[string]string),
		blacklist:          make(map[string]bool),
	}
}

type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func (s *JWTService) GenerateTokenPair(username string) (*TokenPair, error) {
	accessToken, err := s.generateToken(username, s.accessTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateToken(username, s.refreshTokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	s.mu.Lock()
	s.refreshTokens[refreshToken] = username
	s.mu.Unlock()

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *JWTService) ValidateAccessToken(tokenString string) (string, error) {
	s.mu.RLock()
	if s.blacklist[tokenString] {
		s.mu.RUnlock()
		return "", fmt.Errorf("token has been revoked")
	}
	s.mu.RUnlock()

	return s.parseToken(tokenString)
}

// Refresh validates a refresh token, revokes it, and issues a new pair (Rotation).
func (s *JWTService) Refresh(refreshToken string) (*TokenPair, error) {
	s.mu.Lock()
	username, exists := s.refreshTokens[refreshToken]
	if !exists {
		s.mu.Unlock()
		return nil, fmt.Errorf("invalid or expired refresh token")
	}
	// Revoke old refresh token
	delete(s.refreshTokens, refreshToken)
	s.mu.Unlock()

	// Validate the token is not expired
	if _, err := s.parseToken(refreshToken); err != nil {
		return nil, fmt.Errorf("refresh token expired: %w", err)
	}

	return s.GenerateTokenPair(username)
}

// Logout blacklists the access token and revokes the refresh token.
func (s *JWTService) Logout(accessToken, refreshToken string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if accessToken != "" {
		s.blacklist[accessToken] = true
	}
	if refreshToken != "" {
		delete(s.refreshTokens, refreshToken)
	}
}

func (s *JWTService) generateToken(username string, expiry time.Duration) (string, error) {
	jti := make([]byte, 16)
	rand.Read(jti)
	claims := jwt.MapClaims{
		"sub": username,
		"jti": hex.EncodeToString(jti),
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(expiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *JWTService) parseToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	username, ok := claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("invalid token claims")
	}

	return username, nil
}
