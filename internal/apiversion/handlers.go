package apiversion

import (
	"encoding/json"
	"net/http"
)

// VersionsHandler responds with a JSON discovery document listing supported
// API versions, the current default version, and deprecation/sunset details.
func VersionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"current":  V1,
		"versions": Supported,
	})
}
