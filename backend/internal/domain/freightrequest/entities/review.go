package entities

import (
	"time"

	"github.com/google/uuid"
)

// Review represents a review left by one party about the other after freight completion.
type Review struct {
	id            uuid.UUID
	reviewerOrgID uuid.UUID
	rating        int
	comment       string
	createdAt     time.Time
}

// NewReview creates a new Review entity.
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

func (r Review) ID() uuid.UUID            { return r.id }
func (r Review) ReviewerOrgID() uuid.UUID { return r.reviewerOrgID }
func (r Review) Rating() int              { return r.rating }
func (r Review) Comment() string          { return r.comment }
func (r Review) CreatedAt() time.Time     { return r.createdAt }

// EditWindow is the duration during which a review can be edited.
const EditWindow = 24 * time.Hour

// CanEdit returns true if the review can still be edited (within 24h of creation).
func (r Review) CanEdit() bool {
	return time.Since(r.createdAt) <= EditWindow
}

// EditExpiresAt returns the time when the edit window expires.
func (r Review) EditExpiresAt() time.Time {
	return r.createdAt.Add(EditWindow)
}

// TimeUntilEditExpires returns the remaining time for editing.
func (r Review) TimeUntilEditExpires() time.Duration {
	remaining := EditWindow - time.Since(r.createdAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// WithUpdatedRatingAndComment creates a copy of the review with updated rating and comment.
// This preserves the original createdAt timestamp.
func (r Review) WithUpdatedRatingAndComment(rating int, comment string) Review {
	return Review{
		id:            r.id,
		reviewerOrgID: r.reviewerOrgID,
		rating:        rating,
		comment:       comment,
		createdAt:     r.createdAt,
	}
}
