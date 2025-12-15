package entities

import (
	"time"

	"github.com/google/uuid"
)

type Review struct {
	id            uuid.UUID
	reviewerOrgID uuid.UUID
	rating        int
	comment       string
	createdAt     time.Time
}

func NewReview(
	id uuid.UUID,
	reviewerOrgID uuid.UUID,
	rating int,
	comment string,
	createdAt time.Time,
) Review {
	return Review{
		id:            id,
		reviewerOrgID: reviewerOrgID,
		rating:        rating,
		comment:       comment,
		createdAt:     createdAt,
	}
}

func (r Review) ID() uuid.UUID          { return r.id }
func (r Review) ReviewerOrgID() uuid.UUID { return r.reviewerOrgID }
func (r Review) Rating() int            { return r.rating }
func (r Review) Comment() string        { return r.comment }
func (r Review) CreatedAt() time.Time   { return r.createdAt }
