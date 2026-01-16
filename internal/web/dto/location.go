package dto

type LocationRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	UID       string  `json:"uid" binding:"required"`
}
