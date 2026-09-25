package entity

import (
	"time"
)

// Media represents a binary media record stored in PostgreSQL.
type Media struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	FileName  string    `json:"file_name" db:"file_name"`
	MediaType string    `json:"media_type" db:"media_type"`
	FileSize  int64     `json:"file_size" db:"file_size"`
	Data      []byte    `json:"-" db:"data"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TableName specifies the database table for Media.
func (Media) TableName() string {
	return "media_binary"
}

// MediaInfo represents media metadata without loading the large bytea payload into memory.
type MediaInfo struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	FileName  string    `json:"file_name" db:"file_name"`
	MediaType string    `json:"media_type" db:"media_type"`
	FileSize  int64     `json:"file_size" db:"file_size"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TableName specifies the database table for MediaInfo.
func (MediaInfo) TableName() string {
	return "media_binary"
}
