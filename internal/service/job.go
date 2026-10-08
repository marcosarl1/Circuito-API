package service

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

const (
	JobStatusQueued   = "queued"
	JobStatusRunning  = "running"
	JobStatusComplete = "complete"
	JobStatusFailed   = "failed"
)

type ScrapeJob struct {
	JobID      string  `bson:"_id" json:"job_id"`
	Status     string  `bson:"status" json:"status"`
	StartedAt  string  `bson:"started_at" json:"started_at"`
	FinishedAt string  `bson:"finished_at" json:"finished_at"`
	Report     any     `bson:"report" json:"report"`
	Error      *string `bson:"error" json:"error"`
	Active     bool    `bson:"active" json:"-"`
}

func NowISO() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000000+00:00")
}

func NewScrapeJobID() string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return hex.EncodeToString([]byte(NowISO()))[:32]
	}
	return hex.EncodeToString(raw)
}
