package domain

import (
	"time"

	"github.com/google/uuid"
)

type Location struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	CreatedAT time.Time `json:"created_at"`
	InDanger  bool      `json:"in_dangerous_area"`
}

func NewLocation(userID uuid.UUID, latitude, longitude float64) *Location {
	return &Location{
		ID:        uuid.New(),
		UserID:    userID,
		Latitude:  latitude,
		Longitude: longitude,
		CreatedAT: time.Now(),
		InDanger:  false,
	}
}
