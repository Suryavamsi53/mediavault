package auth

import (
	routing "github.com/go-ozzo/ozzo-routing/v2"
	"mediavault/internal/errors"
	"mediavault/pkg/log"
)

// RegisterHandlers registers handlers for authentication requests.
func RegisterHandlers(rg *routing.RouteGroup, service Service, authHandler routing.Handler, logger log.Logger) {
	rg.Post("/login", login(service, logger))
	rg.Post("/register", register(service, logger))

	if authHandler != nil {
		g := rg.Group("")
		g.Use(authHandler)
		g.Get("/me", me())
	}
}

// register handles user registration.
func register(service Service, logger log.Logger) routing.Handler {
	return func(c *routing.Context) error {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := c.Read(&req); err != nil {
			logger.With(c.Request.Context()).Errorf("invalid request: %v", err)
			return errors.BadRequest("Invalid request body")
		}

		token, err := service.Register(c.Request.Context(), req.Username, req.Password)
		if err != nil {
			return err
		}
		return c.Write(struct {
			Token    string `json:"token"`
			Username string `json:"username"`
		}{token, req.Username})
	}
}

// login returns a handler that handles user login request.
func login(service Service, logger log.Logger) routing.Handler {
	return func(c *routing.Context) error {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := c.Read(&req); err != nil {
			logger.With(c.Request.Context()).Errorf("invalid request: %v", err)
			return errors.BadRequest("")
		}

		token, err := service.Login(c.Request.Context(), req.Username, req.Password)
		if err != nil {
			return err
		}
		return c.Write(struct {
			Token    string `json:"token"`
			Username string `json:"username"`
		}{token, req.Username})
	}
}

// me returns the profile of the currently authenticated user.
func me() routing.Handler {
	return func(c *routing.Context) error {
		user := CurrentUser(c.Request.Context())
		if user == nil {
			return errors.Unauthorized("")
		}
		return c.Write(struct {
			ID       string `json:"id"`
			Username string `json:"username"`
		}{user.GetID(), user.GetName()})
	}
}
