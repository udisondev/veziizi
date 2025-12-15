package entities

import (
	"time"

	"github.com/google/uuid"
)

type Document struct {
	id         uuid.UUID
	name       string
	mimeType   string // detected via http.DetectContentType() on upload
	size       int64
	fileID     uuid.UUID
	uploadedBy uuid.UUID
	createdAt  time.Time
}

func NewDocument(
	id uuid.UUID,
	name string,
	mimeType string,
	size int64,
	fileID uuid.UUID,
	uploadedBy uuid.UUID,
	createdAt time.Time,
) Document {
	return Document{
		id:         id,
		name:       name,
		mimeType:   mimeType,
		size:       size,
		fileID:     fileID,
		uploadedBy: uploadedBy,
		createdAt:  createdAt,
	}
}

func (d Document) ID() uuid.UUID         { return d.id }
func (d Document) Name() string          { return d.name }
func (d Document) MimeType() string      { return d.mimeType }
func (d Document) Size() int64           { return d.size }
func (d Document) FileID() uuid.UUID     { return d.fileID }
func (d Document) UploadedBy() uuid.UUID { return d.uploadedBy }
func (d Document) CreatedAt() time.Time  { return d.createdAt }
