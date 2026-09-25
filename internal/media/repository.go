package media

import (
	"context"

	"github.com/go-ozzo/ozzo-dbx"
	"mediavault/internal/entity"
	"mediavault/pkg/dbcontext"
	"mediavault/pkg/log"
)

// Repository encapsulates the logic to access binary media from the database.
type Repository interface {
	// Get returns the media with the specified ID and owner userID including raw binary.
	Get(ctx context.Context, id, userID string) (entity.Media, error)
	// GetInfo returns only media metadata for specified ID and owner userID.
	GetInfo(ctx context.Context, id, userID string) (entity.MediaInfo, error)
	// Count returns the total number of media records belonging to the user.
	Count(ctx context.Context, userID string) (int, error)
	// Query returns the list of media metadata for the user with given offset and limit.
	Query(ctx context.Context, userID string, offset, limit int) ([]entity.MediaInfo, error)
	// Create saves a new media binary record with user ownership in the database.
	Create(ctx context.Context, media entity.Media) error
	// Delete removes the media record belonging to the user.
	Delete(ctx context.Context, id, userID string) error
}

type repository struct {
	db     *dbcontext.DB
	logger log.Logger
}

// NewRepository creates a new media repository.
func NewRepository(db *dbcontext.DB, logger log.Logger) Repository {
	return repository{db, logger}
}

// Get reads the media record with binary data from the database, scoped to owner.
func (r repository) Get(ctx context.Context, id, userID string) (entity.Media, error) {
	var media entity.Media
	q := r.db.With(ctx).Select().From("media_binary").Where(dbx.HashExp{"id": id})
	if userID != "" {
		q = q.AndWhere(dbx.HashExp{"user_id": userID})
	}
	err := q.One(&media)
	return media, err
}

// GetInfo reads media metadata without loading the binary payload into memory.
func (r repository) GetInfo(ctx context.Context, id, userID string) (entity.MediaInfo, error) {
	var info entity.MediaInfo
	q := r.db.With(ctx).
		Select("id", "user_id", "file_name", "media_type", "file_size", "created_at", "updated_at").
		From("media_binary").
		Where(dbx.HashExp{"id": id})
	if userID != "" {
		q = q.AndWhere(dbx.HashExp{"user_id": userID})
	}
	err := q.One(&info)
	return info, err
}

// Count returns the number of media records for the user.
func (r repository) Count(ctx context.Context, userID string) (int, error) {
	var count int
	q := r.db.With(ctx).Select("COUNT(*)").From("media_binary")
	if userID != "" {
		q = q.Where(dbx.HashExp{"user_id": userID})
	}
	err := q.Row(&count)
	return count, err
}

// Query retrieves media metadata for the user with specified offset and limit.
func (r repository) Query(ctx context.Context, userID string, offset, limit int) ([]entity.MediaInfo, error) {
	var list []entity.MediaInfo
	q := r.db.With(ctx).
		Select("id", "user_id", "file_name", "media_type", "file_size", "created_at", "updated_at").
		From("media_binary")
	if userID != "" {
		q = q.Where(dbx.HashExp{"user_id": userID})
	}
	err := q.OrderBy("created_at DESC").
		Offset(int64(offset)).
		Limit(int64(limit)).
		All(&list)
	return list, err
}

// Create inserts a new media record into the database.
func (r repository) Create(ctx context.Context, media entity.Media) error {
	return r.db.With(ctx).Model(&media).Insert()
}

// Delete removes a media record belonging to the user.
func (r repository) Delete(ctx context.Context, id, userID string) error {
	media, err := r.Get(ctx, id, userID)
	if err != nil {
		return err
	}
	return r.db.With(ctx).Model(&media).Delete()
}
