package media

import (
	"context"
	"fmt"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"mediavault/internal/entity"
	"mediavault/pkg/log"
)

// MediaInfo represents media metadata along with an access URL.
type MediaInfo struct {
	entity.MediaInfo
	DownloadURL string `json:"download_url"`
}

// Service encapsulates usecase logic for binary media.
type Service interface {
	Get(ctx context.Context, id, userID string) (entity.Media, error)
	GetInfo(ctx context.Context, id, userID string) (MediaInfo, error)
	Query(ctx context.Context, userID string, offset, limit int) ([]MediaInfo, error)
	Count(ctx context.Context, userID string) (int, error)
	Upload(ctx context.Context, userID, fileName, mediaType string, data []byte) (MediaInfo, error)
	Delete(ctx context.Context, id, userID string) (MediaInfo, error)
}

type service struct {
	repo   Repository
	logger log.Logger
}

// NewService creates a new media service.
func NewService(repo Repository, logger log.Logger) Service {
	return service{repo, logger}
}

// Get returns the media with full binary data.
func (s service) Get(ctx context.Context, id, userID string) (entity.Media, error) {
	return s.repo.Get(ctx, id, userID)
}

// GetInfo returns the media metadata with download URL.
func (s service) GetInfo(ctx context.Context, id, userID string) (MediaInfo, error) {
	info, err := s.repo.GetInfo(ctx, id, userID)
	if err != nil {
		return MediaInfo{}, err
	}
	return MediaInfo{
		MediaInfo:   info,
		DownloadURL: fmt.Sprintf("/v1/media/%s/content", info.ID),
	}, nil
}

// Count returns the total count of media records for the user.
func (s service) Count(ctx context.Context, userID string) (int, error) {
	return s.repo.Count(ctx, userID)
}

// Query returns paginated media metadata for the user.
func (s service) Query(ctx context.Context, userID string, offset, limit int) ([]MediaInfo, error) {
	items, err := s.repo.Query(ctx, userID, offset, limit)
	if err != nil {
		return nil, err
	}
	result := make([]MediaInfo, len(items))
	for i, item := range items {
		result[i] = MediaInfo{
			MediaInfo:   item,
			DownloadURL: fmt.Sprintf("/v1/media/%s/content", item.ID),
		}
	}
	return result, nil
}

// Upload stores a new binary media file associated with the user in the database.
func (s service) Upload(ctx context.Context, userID, fileName, mediaType string, data []byte) (MediaInfo, error) {
	if err := validation.Validate(fileName, validation.Required, validation.Length(1, 255)); err != nil {
		return MediaInfo{}, err
	}
	if len(data) == 0 {
		return MediaInfo{}, validation.NewError("validation_file_empty", "file content cannot be empty")
	}

	id := entity.GenerateID()
	now := time.Now()
	media := entity.Media{
		ID:        id,
		UserID:    userID,
		FileName:  fileName,
		MediaType: mediaType,
		FileSize:  int64(len(data)),
		Data:      data,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, media); err != nil {
		return MediaInfo{}, err
	}

	return s.GetInfo(ctx, id, userID)
}

// Delete removes the media record.
func (s service) Delete(ctx context.Context, id, userID string) (MediaInfo, error) {
	info, err := s.GetInfo(ctx, id, userID)
	if err != nil {
		return MediaInfo{}, err
	}
	if err := s.repo.Delete(ctx, id, userID); err != nil {
		return MediaInfo{}, err
	}
	return info, nil
}
