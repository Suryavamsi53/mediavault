package auth

import (
	"context"
	"mediavault/internal/entity"
	"mediavault/internal/errors"
	"mediavault/pkg/log"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_service_Authenticate(t *testing.T) {
	logger, _ := log.NewForTest()
	s := NewService(nil, "test", 100, logger)
	_, err := s.Login(context.Background(), "unknown", "bad")
	assert.Equal(t, errors.Unauthorized("Invalid username or password"), err)
	token, err := s.Login(context.Background(), "demo", "pass")
	assert.Nil(t, err)
	assert.NotEmpty(t, token)
}

func Test_service_authenticate(t *testing.T) {
	logger, _ := log.NewForTest()
	s := service{db: nil, signingKey: "test", tokenExpiration: 100, logger: logger}
	assert.Nil(t, s.authenticate(context.Background(), "unknown", "bad"))
	assert.NotNil(t, s.authenticate(context.Background(), "demo", "pass"))
}

func Test_service_GenerateJWT(t *testing.T) {
	logger, _ := log.NewForTest()
	s := service{db: nil, signingKey: "test", tokenExpiration: 100, logger: logger}
	token, err := s.generateJWT(entity.User{
		ID:   "100",
		Name: "demo",
	})
	if assert.Nil(t, err) {
		assert.NotEmpty(t, token)
	}
}
