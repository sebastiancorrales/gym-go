package security

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("token has expired")
	ErrInvalidTokenType = errors.New("invalid token type")
)

// TokenType represents the type of JWT token
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// Claims represents JWT claims
type Claims struct {
	UserID    uuid.UUID `json:"user_id"`
	GymID     uuid.UUID `json:"gym_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	TokenType TokenType `json:"token_type"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT operations
type JWTManager struct {
	accessSecret      string
	refreshSecret     string
	accessExpiration  time.Duration
	refreshExpiration time.Duration
	issuer            string
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(accessSecret, refreshSecret string, accessExp, refreshExp time.Duration, issuer string) *JWTManager {
	return &JWTManager{
		accessSecret:      accessSecret,
		refreshSecret:     refreshSecret,
		accessExpiration:  accessExp,
		refreshExpiration: refreshExp,
		issuer:            issuer,
	}
}

// GenerateAccessToken generates an access token
func (j *JWTManager) GenerateAccessToken(userID, gymID uuid.UUID, email, role string) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:    userID,
		GymID:     gymID,
		Email:     email,
		Role:      role,
		TokenType: AccessToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(j.accessExpiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    j.issuer,
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.accessSecret))
}

// GenerateRefreshToken generates a refresh token
func (j *JWTManager) GenerateRefreshToken(userID, gymID uuid.UUID, email, role string) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:    userID,
		GymID:     gymID,
		Email:     email,
		Role:      role,
		TokenType: RefreshToken,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(j.refreshExpiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    j.issuer,
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.refreshSecret))
}

// ValidateAccessToken validates an access token and returns claims
func (j *JWTManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	return j.validateToken(tokenString, j.accessSecret, AccessToken)
}

// ValidateRefreshToken validates a refresh token and returns claims
func (j *JWTManager) ValidateRefreshToken(tokenString string) (*Claims, error) {
	return j.validateToken(tokenString, j.refreshSecret, RefreshToken)
}

// validateToken validates a token with the given secret and type
func (j *JWTManager) validateToken(tokenString, secret string, expectedType TokenType) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.TokenType != expectedType {
		return nil, ErrInvalidTokenType
	}

	return claims, nil
}

// ExtractUserID extracts user ID from token without full validation
func (j *JWTManager) ExtractUserID(tokenString string) (uuid.UUID, error) {
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return uuid.Nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return uuid.Nil, ErrInvalidToken
	}

	return claims.UserID, nil
}



