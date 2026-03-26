package domain

import (
	"time"

	"github.com/google/uuid"
)

// Location represents a user location check record.
type Location struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	CreatedAt time.Time `json:"created_at"`
	InDanger  bool      `json:"in_dangerous_area"`
}

// NewLocation creates a new Location with a generated ID and current timestamp.
func NewLocation(userID uuid.UUID, latitude, longitude float64) *Location {
	return &Location{
		ID:        uuid.New(),
		UserID:    userID,
		Latitude:  latitude,
		Longitude: longitude,
		CreatedAt: time.Now(),
		InDanger:  false,
	}
}
