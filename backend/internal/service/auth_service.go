package service

import (
	"context"

	"pebt/backend/internal/auth"
)

// Register validates the account fields and creates the user. The session
// token itself is issued by the HTTP layer (which owns the secret and cookie).
func (s *Service) Register(ctx context.Context, emailRaw, passwordRaw Text) (int64, *Error) {
	email, e := validateEmail(emailRaw)
	if e != nil {
		return 0, e
	}
	password, e := validatePassword(passwordRaw)
	if e != nil {
		return 0, e
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return 0, internal(s.Log, err)
	}
	id, err := s.Store.CreateUser(ctx, email, hash)
	if err != nil {
		return 0, s.mapDBError(err, "")
	}
	return id, nil
}

// Login verifies credentials. Unknown email and wrong password return the
// identical error so nothing about account existence leaks (D28).
func (s *Service) Login(ctx context.Context, emailRaw, passwordRaw Text) (int64, *Error) {
	user, err := s.Store.GetUserByEmail(ctx, string(emailRaw))
	if err != nil {
		return 0, internal(s.Log, err)
	}
	invalid := errf(401, "INVALID_CREDENTIALS", "Invalid email or password")
	if user == nil || !auth.CheckPassword(user.PasswordHash, string(passwordRaw)) {
		return 0, invalid
	}
	return user.ID, nil
}
