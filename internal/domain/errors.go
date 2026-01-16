package domain

import "errors"

var (
	ErrInvalidTitle       = errors.New("invalid title: must be between 5 and 100 characters")
	ErrInvalidDescription = errors.New("invalid description: must be between 10 and 500 characters")
	ErrInvalidLatitude    = errors.New("invalid latitude: must be between -90 and 90")
	ErrInvalidLongitude   = errors.New("invalid longitude: must be between -180 and 180")
	ErrInvalidRadius      = errors.New("invalid radius: must be a positive number")
	ErrInvalidSeverity    = errors.New("invalid severity: must be one of 'low', 'medium', 'high'")
	ErrInvalidType        = errors.New("invalid type: must not be empty")
	ErrIncidentNotFound   = errors.New("incident not found")
	ErrInvalidID          = errors.New("invalid ID format")
)
