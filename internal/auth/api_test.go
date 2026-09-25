package auth

import (
	"context"
	"mediavault/internal/errors"
	"mediavault/internal/test"
	"mediavault/pkg/log"
	"net/http"
	"testing"
)

type mockService struct{}

func (m mockService) Login(ctx context.Context, username, password string) (string, error) {
	if username == "test" && password == "pass" {
		return "token-100", nil
	}
	return "", errors.Unauthorized("")
}

func (m mockService) Register(ctx context.Context, username, password string) (string, error) {
	if username == "newuser" && password == "pass" {
		return "token-200", nil
	}
	return "", errors.BadRequest("invalid")
}

func TestAPI(t *testing.T) {
	logger, _ := log.NewForTest()
	router := test.MockRouter(logger)
	RegisterHandlers(router.Group(""), mockService{}, MockAuthHandler, logger)

	header := MockAuthHeader()

	tests := []test.APITestCase{
		{"success", "POST", "/login", `{"username":"test","password":"pass"}`, nil, http.StatusOK, `{"token":"token-100","username":"test"}`},
		{"bad credential", "POST", "/login", `{"username":"test","password":"wrong pass"}`, nil, http.StatusUnauthorized, ""},
		{"bad json", "POST", "/login", `{"username":"test"`, nil, http.StatusBadRequest, ""},
		{"register success", "POST", "/register", `{"username":"newuser","password":"pass"}`, nil, http.StatusOK, `{"token":"token-200","username":"newuser"}`},
		{"me success", "GET", "/me", "", header, http.StatusOK, `{"id":"100","username":"Tester"}`},
		{"me unauthorized", "GET", "/me", "", nil, http.StatusUnauthorized, ""},
	}
	for _, tc := range tests {
		test.Endpoint(t, router, tc)
	}
}
