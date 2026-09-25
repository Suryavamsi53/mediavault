package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-ozzo/ozzo-dbx"
	"mediavault/internal/entity"
	"mediavault/internal/errors"
	"mediavault/pkg/dbcontext"
	"mediavault/pkg/log"
)

// Service encapsulates the authentication logic.
type Service interface {
	Login(ctx context.Context, username, password string) (string, error)
	Register(ctx context.Context, username, password string) (string, error)
}

// Identity represents an authenticated user identity.
type Identity interface {
	GetID() string
	GetName() string
}

type service struct {
	db              *dbcontext.DB
	signingKey      string
	tokenExpiration int
	logger          log.Logger
}

// NewService creates a new authentication service.
func NewService(db *dbcontext.DB, signingKey string, tokenExpiration int, logger log.Logger) Service {
	return service{db, signingKey, tokenExpiration, logger}
}

func hashPassword(password string) string {
	salt := "MediaVault2026Salt"
	h := sha256.Sum256([]byte(password + ":" + salt))
	return fmt.Sprintf("%x", h)
}

// Register creates a new user in PostgreSQL and returns a valid JWT token.
func (s service) Register(ctx context.Context, username, password string) (string, error) {
	if len(username) < 3 || len(password) < 4 {
		return "", errors.BadRequest("Username must be at least 3 characters and password at least 4 characters")
	}

	var count int
	err := s.db.With(ctx).Select("COUNT(*)").From("users").Where(dbx.HashExp{"username": username}).Row(&count)
	if err != nil && err != sql.ErrNoRows {
		s.logger.With(ctx).Errorf("failed checking username existence: %v", err)
		return "", errors.InternalServerError("")
	}
	if count > 0 {
		return "", errors.BadRequest("Username already exists. Please choose a different one.")
	}

	id := entity.GenerateID()
	user := entity.User{
		ID:        id,
		Name:      username,
		Password:  hashPassword(password),
		CreatedAt: time.Now(),
	}

	if err := s.db.With(ctx).Model(&user).Insert(); err != nil {
		s.logger.With(ctx).Errorf("failed creating user: %v", err)
		return "", errors.InternalServerError("Failed to create account")
	}

	return s.generateJWT(user)
}

// authenticate authenticates a user using username and password.
// If username and password are correct, an identity is returned. Otherwise, nil is returned.
func (s service) authenticate(ctx context.Context, username, password string) Identity {
	logger := s.logger.With(ctx, "user", username)

	if s.db != nil {
		var user entity.User
		err := s.db.With(ctx).Select().From("users").Where(dbx.HashExp{"username": username}).One(&user)
		if err == nil {
			if user.Password == hashPassword(password) {
				logger.Infof("authentication successful")
				return user
			}
		}
	}

	// Demo fallback
	if username == "demo" && password == "pass" {
		logger.Infof("demo authentication successful")
		return entity.User{ID: "demo-user-id", Name: "demo"}
	}

	logger.Infof("authentication failed")
	return nil
}

// Login authenticates a user and generates a JWT token if authentication succeeds.
func (s service) Login(ctx context.Context, username, password string) (string, error) {
	if identity := s.authenticate(ctx, username, password); identity != nil {
		return s.generateJWT(identity)
	}
	return "", errors.Unauthorized("Invalid username or password")
}

// generateJWT generates a JWT that encodes an identity.
func (s service) generateJWT(identity Identity) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":   identity.GetID(),
		"name": identity.GetName(),
		"exp":  time.Now().Add(time.Duration(s.tokenExpiration) * time.Hour).Unix(),
	}).SignedString([]byte(s.signingKey))
}
