package domain

import (
	"time"

	"github.com/google/uuid"
)

type SeverityLevel string

const (
	SeverityLow    SeverityLevel = "low"
	SeverityMedium SeverityLevel = "medium"
	SeverityHigh   SeverityLevel = "high"
)

type Incident struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`

	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Radius    float64 `json:"radius"`

	Severity SeverityLevel `json:"severity"`
	Type     string        `json:"type"`

	IsActive  bool
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewIncident(title, description string, latitude, longitude, radius float64, severity SeverityLevel, incidentType string) *Incident {
	return &Incident{
		ID:          uuid.New(),
		Title:       title,
		Description: description,
		Latitude:    latitude,
		Longitude:   longitude,
		Radius:      radius,
		Severity:    severity,
		Type:        incidentType,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func (i *Incident) Update(title, description string, latitude, longitude, radius float64, severity SeverityLevel, incidentType string) {
	i.Title = title
	i.Description = description
	i.Latitude = latitude
	i.Longitude = longitude
	i.Radius = radius
	i.Severity = severity
	i.Type = incidentType
	i.UpdatedAt = time.Now()
}
