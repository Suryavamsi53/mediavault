package media

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-ozzo/ozzo-routing/v2"
	"mediavault/internal/auth"
	"mediavault/internal/errors"
	"mediavault/pkg/log"
	"mediavault/pkg/pagination"
)

const maxUploadSize = 25 << 20 // 25 MB max limit for database BYTEA storage

// Allowed MIME types whitelist
var allowedMIMETypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	"image/webp":      true,
	"image/bmp":       true,
	"video/mp4":       true,
	"video/webm":      true,
	"video/ogg":       true,
	"video/quicktime": true,
	"audio/mpeg":      true,
	"audio/ogg":       true,
	"audio/wav":       true,
}

// Allowed file extensions whitelist
var allowedExtensions = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".bmp":  "image/bmp",
	".mp4":  "video/mp4",
	".webm": "video/webm",
	".ogg":  "video/ogg",
	".mov":  "video/quicktime",
	".mp3":  "audio/mpeg",
	".wav":  "audio/wav",
}

// RegisterHandlers registers the media endpoints into the routing tree.
func RegisterHandlers(r *routing.RouteGroup, service Service, authHandler routing.Handler, signingKey string, logger log.Logger) {
	res := resource{service, signingKey, logger}

	// Streaming endpoint: supports Authorization header OR ?token=<jwt> for HTML5 video/img tags
	r.Get("/media/<id>/content", res.getContent)

	// Protected routes require authentication
	r.Use(authHandler)
	r.Get("/media", res.query)
	r.Get("/media/<id>", res.getInfo)
	r.Post("/media/upload", res.upload)
	r.Delete("/media/<id>", res.delete)
}

type resource struct {
	service    Service
	signingKey string
	logger     log.Logger
}

func (r resource) query(c *routing.Context) error {
	user := auth.CurrentUser(c.Request.Context())
	if user == nil {
		return errors.Unauthorized("")
	}

	ctx := c.Request.Context()
	count, err := r.service.Count(ctx, user.GetID())
	if err != nil {
		return err
	}
	pages := pagination.NewFromRequest(c.Request, count)
	items, err := r.service.Query(ctx, user.GetID(), pages.Offset(), pages.Limit())
	if err != nil {
		return err
	}
	pages.Items = items
	return c.Write(pages)
}

func (r resource) getInfo(c *routing.Context) error {
	user := auth.CurrentUser(c.Request.Context())
	if user == nil {
		return errors.Unauthorized("")
	}

	id := c.Param("id")
	info, err := r.service.GetInfo(c.Request.Context(), id, user.GetID())
	if err != nil {
		return errors.NotFound("media not found")
	}
	return c.Write(info)
}

func (r resource) getContent(c *routing.Context) error {
	var userID string
	if user := auth.CurrentUser(c.Request.Context()); user != nil {
		userID = user.GetID()
	} else {
		// Extract token from query parameter: ?token=...
		tokenStr := c.Query("token")
		if tokenStr == "" {
			// Check Authorization header
			authHeader := c.Request.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenStr == "" {
			return errors.Unauthorized("authentication token required to stream media")
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(r.signingKey), nil
		})
		if err != nil || !token.Valid {
			return errors.Unauthorized("invalid or expired token")
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || claims["id"] == nil {
			return errors.Unauthorized("invalid token claims")
		}
		userID = claims["id"].(string)
	}

	id := c.Param("id")
	media, err := r.service.Get(c.Request.Context(), id, userID)
	if err != nil {
		return errors.NotFound("media not found or access denied")
	}

	c.Response.Header().Set("X-Content-Type-Options", "nosniff")
	c.Response.Header().Set("X-Frame-Options", "SAMEORIGIN")
	c.Response.Header().Set("Content-Security-Policy", "default-src 'none'; media-src 'self'; style-src 'unsafe-inline'; sandbox allow-downloads")

	c.Response.Header().Set("Content-Type", media.MediaType)
	c.Response.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, media.FileName))
	c.Response.Header().Set("Accept-Ranges", "bytes")

	reader := bytes.NewReader(media.Data)
	http.ServeContent(c.Response, c.Request, media.FileName, media.UpdatedAt, reader)
	return nil
}

func (r resource) upload(c *routing.Context) error {
	user := auth.CurrentUser(c.Request.Context())
	if user == nil {
		return errors.Unauthorized("")
	}

	c.Request.Body = http.MaxBytesReader(c.Response, c.Request.Body, maxUploadSize)
	if err := c.Request.ParseMultipartForm(maxUploadSize); err != nil {
		r.logger.With(c.Request.Context()).Errorf("failed to parse multipart form: %v", err)
		return errors.BadRequest("file size exceeds the 25MB limit or invalid multipart form")
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		r.logger.With(c.Request.Context()).Infof("missing form file: %v", err)
		return errors.BadRequest("missing 'file' field in multipart form")
	}
	defer file.Close()

	cleanFileName := filepath.Base(filepath.Clean(header.Filename))
	if cleanFileName == "." || cleanFileName == "/" || cleanFileName == "" {
		cleanFileName = "unnamed_media"
	}

	ext := strings.ToLower(filepath.Ext(cleanFileName))
	expectedMIME, hasAllowedExt := allowedExtensions[ext]
	if !hasAllowedExt {
		return errors.BadRequest("unsupported file extension: only images and videos are permitted")
	}

	data, err := io.ReadAll(file)
	if err != nil {
		r.logger.With(c.Request.Context()).Errorf("failed reading uploaded file: %v", err)
		return errors.InternalServerError("failed to read uploaded file")
	}

	sniffLen := 512
	if len(data) < sniffLen {
		sniffLen = len(data)
	}
	detectedType := http.DetectContentType(data[:sniffLen])

	finalMIME := expectedMIME
	if allowedMIMETypes[detectedType] {
		finalMIME = detectedType
	} else if detectedType == "application/octet-stream" {
		finalMIME = expectedMIME
	} else if !allowedMIMETypes[finalMIME] {
		return errors.BadRequest("file content does not match an allowed image or video format")
	}

	info, err := r.service.Upload(c.Request.Context(), user.GetID(), cleanFileName, finalMIME, data)
	if err != nil {
		return err
	}

	c.Response.WriteHeader(http.StatusCreated)
	return c.Write(info)
}

func (r resource) delete(c *routing.Context) error {
	user := auth.CurrentUser(c.Request.Context())
	if user == nil {
		return errors.Unauthorized("")
	}

	id := c.Param("id")
	info, err := r.service.Delete(c.Request.Context(), id, user.GetID())
	if err != nil {
		return errors.NotFound("media not found")
	}
	return c.Write(info)
}
