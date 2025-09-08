package telemetry

import (
	"encoding/json"
	"net/http"
	"regexp"
	"time"
)

type Payload struct {
	Type    string            `json:"type"`
	Message string            `json:"message"`
	Tags    map[string]string `json:"tags"`
	TS      time.Time         `json:"ts"`
}

var re = regexp.MustCompile(`(?i)(bearer\s+[A-Za-z0-9._-]+|api[-_]?key\s*[=:]\s*[A-Za-z0-9._-]+|token\s*[=:]\s*[A-Za-z0-9._-]+)`)

func mask(s string) string { return re.ReplaceAllString(s, "***redacted***") }

func Handle(w http.ResponseWriter, r *http.Request) {
	var p Payload
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if p.Tags == nil {
		p.Tags = map[string]string{}
	}
	p.Message = mask(p.Message)
	for k, v := range p.Tags {
		p.Tags[k] = mask(v)
	}
	if p.TS.IsZero() {
		p.TS = time.Now().UTC()
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}
