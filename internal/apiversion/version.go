package apiversion

import "time"

// API version constants.
const (
	V1 = "v1"
	V2 = "v2"
)

// Status describes the lifecycle state of an API version.
type Status string

const (
	// StatusStable indicates the version is the current, supported version.
	StatusStable Status = "stable"
	// StatusDeprecated indicates the version is still available but scheduled
	// for removal.
	StatusDeprecated Status = "deprecated"
)

// Sunset is the future date at which deprecated API versions are planned to be
// removed. It is exposed so callers can reference the same date in headers and
// documentation.
var Sunset = time.Date(2027, time.December, 31, 23, 59, 59, 0, time.UTC)

// VersionInfo describes a supported API version for discovery responses.
type VersionInfo struct {
	Version    string     `json:"version"`
	Status     Status     `json:"status"`
	Path       string     `json:"path"`
	Deprecated bool       `json:"deprecated"`
	Current    bool       `json:"current"`
	SunsetDate *time.Time `json:"sunset_date,omitempty"`
}

// Supported lists the API versions offered by the application.
var Supported = []VersionInfo{
	{
		Version: V1,
		Status:  StatusStable,
		Path:    "/api/" + V1,
		Current: true,
	},
	{
		Version:    V2,
		Status:     StatusDeprecated,
		Path:       "/api/" + V2,
		Deprecated: true,
		SunsetDate: &Sunset,
	},
}
