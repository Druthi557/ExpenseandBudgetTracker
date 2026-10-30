// Package auth implements password hashing (bcrypt cost 12 — D6) and HS256
// JWT sessions (D9/D10): 60-minute tokens with sub/iat/exp claims, delivered
// to clients only via an HttpOnly cookie (never in a response body).
package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	CookieName = "pebt_token"
	Issuer     = "pebt"
	TokenTTL   = time.Hour
	BcryptCost = 12
)

var (
	ErrExpired = errors.New("token expired")
	ErrInvalid = errors.New("token invalid")
)

// Now is the clock used for token timestamps. It is a variable so the test
// suite can mint already-expired tokens (TC-A30) without a real wait.
var Now = time.Now

// HashPassword hashes a password with bcrypt cost 12.
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	return string(b), err
}

// CheckPassword reports whether the password matches the stored hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// IssueToken mints a 60-minute HS256 JWT and returns it with its expiry.
func IssueToken(userID int64, secret []byte) (string, time.Time, error) {
	now := Now()
	exp := now.Add(TokenTTL)
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		Issuer:    Issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(exp),
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	return tok, exp, err
}

// VerifyToken validates signature, algorithm, issuer and expiry, returning
// the authenticated user id.
func VerifyToken(tokenString string, secret []byte) (int64, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (any, error) { return secret, nil },
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(Issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return 0, ErrExpired
		}
		return 0, ErrInvalid
	}
	uid, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || uid <= 0 {
		return 0, ErrInvalid
	}
	return uid, nil
}
