package events

type SLO struct {
	Project string `json:"project,omitempty" yaml:"project,omitempty"`
	Name    string `json:"name,omitempty"    yaml:"name,omitempty"`
}

type Event struct {
	EventStart string `json:"eventStart,omitempty" yaml:"eventStart,omitempty"`
	EventEnd   string `json:"eventEnd,omitempty"   yaml:"eventEnd,omitempty"`
	SLOs       []SLO  `json:"slos,omitempty"       yaml:"slos,omitempty"`
}
