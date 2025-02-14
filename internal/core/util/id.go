package util

import (
	"time"
)

// GenerateID generates a time-based unique identifier
func GenerateID() string {
	return time.Now().Format("20060102150405")
}
