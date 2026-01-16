package domain

type LocationCheckTask struct {
	Location  *Location   `json:"user_location"`
	Incidents []*Incident `json:"incidents"`
}
