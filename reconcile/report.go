package reconcile

import (
	"bytes"
	_ "embed"
	"html/template"
	"time"
)

//go:embed template.txt
var reportTemplateText string

var reportTemplate = template.Must(template.New("report").Parse(reportTemplateText))

type Report struct {
	Date      time.Time          `json:"executionDate"`
	Duration  time.Duration      `json:"executionDuration"`
	User      string             `json:"user"`
	CheckMeIn bool               `json:"checkmein"`
	Discord   map[string]Changes `json:"discord"`
	Groups    map[string]Changes `json:"groups"`
}

type Changes struct {
	Additions []string `json:"add"`
	Deletions []string `json:"delete"`
	Errored   []string `json:"errored,omitempty"`
}

func (r Report) CheckMeInStatus() string {
	if r.CheckMeIn {
		return "SUCCESSFUL"
	}
	return "NOT SUCCESSFUL"
}

func (r Report) RenderText() ([]byte, error) {
	var cache bytes.Buffer
	err := reportTemplate.Execute(&cache, r)
	return cache.Bytes(), err
}

func (r Report) HasChanges() bool {
	// send report if CheckMeIn failed
	if !r.CheckMeIn {
		return true
	}
	for _, d := range r.Discord {
		if len(d.Additions) > 0 || len(d.Deletions) > 0 || len(d.Errored) > 0 {
			return true
		}
	}
	for _, d := range r.Groups {
		if len(d.Additions) > 0 || len(d.Deletions) > 0 || len(d.Errored) > 0 {
			return true
		}
	}

	return false
}
