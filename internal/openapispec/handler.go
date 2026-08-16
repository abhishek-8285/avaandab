package openapispec

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
)

const specFile = "openapi.yaml"

var (
	specBytes []byte
	specErr   error
)

func init() {
	specBytes, specErr = loadSpec()
}

// RegisterRoutes mounts `/openapi.json` and `/openapi.yaml` on the provided router.
func RegisterRoutes(r chi.Router) {
	r.Get("/openapi.json", serveJSON)
	r.Get("/openapi.yaml", serveYAML)
}

// Handler returns an http.Handler that serves `/openapi.json` and `/openapi.yaml`.
func Handler() http.Handler {
	r := chi.NewRouter()
	RegisterRoutes(r)
	return r
}

func loadSpec() ([]byte, error) {
	if b, err := os.ReadFile(specFile); err == nil {
		return b, nil
	}

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return nil, os.ErrNotExist
	}
	dir := filepath.Dir(file)               // internal/openapispec
	root := filepath.Dir(filepath.Dir(dir)) // project root
	return os.ReadFile(filepath.Join(root, specFile))
}

func serveYAML(w http.ResponseWriter, r *http.Request) {
	if specErr != nil {
		http.Error(w, `{"error":"openapi spec not found"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(specBytes)
}

func serveJSON(w http.ResponseWriter, r *http.Request) {
	if specErr != nil {
		http.Error(w, `{"error":"openapi spec not found"}`, http.StatusInternalServerError)
		return
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(specBytes, &doc); err != nil {
		http.Error(w, `{"error":"invalid openapi yaml"}`, http.StatusInternalServerError)
		return
	}

	jsonBytes, err := json.MarshalIndent(cleanYAML(doc), "", "  ")
	if err != nil {
		http.Error(w, `{"error":"failed to serialize openapi json"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(jsonBytes)
}

func cleanYAML(v interface{}) interface{} {
	switch x := v.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{}, len(x))
		for k, v2 := range x {
			ks, _ := k.(string)
			m[ks] = cleanYAML(v2)
		}
		return m
	case map[string]interface{}:
		m := make(map[string]interface{}, len(x))
		for k, v2 := range x {
			m[k] = cleanYAML(v2)
		}
		return m
	case []interface{}:
		a := make([]interface{}, len(x))
		for i, v2 := range x {
			a[i] = cleanYAML(v2)
		}
		return a
	default:
		return x
	}
}
