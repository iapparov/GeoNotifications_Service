package dto

type SystemHealthResponse struct {
	Status string       `json:"status"` // ok / degraded
	Checks HealthChecks `json:"checks"`
}

type HealthChecks struct {
	Database bool `json:"database"`
	Redis    bool `json:"redis"`
}
