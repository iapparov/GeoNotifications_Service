package dto

type IncidentRequestCreate struct {
	Title        string  `json:"title" binding:"required"`
	Description  string  `json:"description" binding:"required"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Radius       float64 `json:"radius"`
	Severity     string  `json:"severity" binding:"required,oneof=low medium high"`
	IncidentType string  `json:"incident_type" binding:"required"`
}

type IncidentRequestUpdate struct {
	Title        string  `json:"title" binding:"required"`
	Description  string  `json:"description" binding:"required"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Radius       float64 `json:"radius"`
	Severity     string  `json:"severity" binding:"required,oneof=low medium high"`
	IncidentType string  `json:"incident_type" binding:"required"`
}
