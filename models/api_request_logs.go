package models

import "time"

type APIRequestLog struct {
	RequestedAt   time.Time
	PasteID       *string
	ClientIP      string
	Method        string
	Route         string
	StatusCode    int
	LatencyUS     int64
	ResponseBytes int64
	UserAgent     *string
}
