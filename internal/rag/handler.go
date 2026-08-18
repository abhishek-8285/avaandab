package rag

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Service exposes the underlying RAG service (for the agent support tool).
func (h *Handler) Service() *Service {
	if h == nil {
		return nil
	}
	return h.service
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/api/rag/search", h.handleSearch)
	r.Post("/api/rag/index", h.handleIndex)
	r.Get("/api/rag/stats", h.handleStats)
	r.Post("/api/rag/reindex", h.handleReindex)
	r.Post("/api/rag/teach", h.handleTeach)
	r.Post("/api/rag/upload", h.handleUpload)
}

type searchRequest struct {
	Query  string `json:"query"`
	TopK   int    `json:"top_k"`
	Source string `json:"source"`
}

type indexRequest struct {
	Directory string `json:"directory"`
}

type APIError struct {
	Error string `json:"error"`
}

func (h *Handler) handleSearch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req searchRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{Error: "failed to read request body"})
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{Error: "invalid JSON"})
		return
	}

	if req.Query == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{Error: "query is required"})
		return
	}

	if req.TopK <= 0 {
		req.TopK = 5
	}

	result, err := h.service.Query(req.Query, req.TopK)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIError{Error: err.Error()})
		return
	}

	// Filter by source if specified
	if req.Source != "" {
		var filtered []VectorEntry
		var filteredScores []float64
		for i, c := range result.Chunks {
			if c.Source == req.Source {
				filtered = append(filtered, c)
				filteredScores = append(filteredScores, result.Scores[i])
			}
		}
		result.Chunks = filtered
		result.Scores = filteredScores
		result.Total = len(filtered)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req indexRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{Error: "failed to read request body"})
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{Error: "invalid JSON"})
		return
	}

	if req.Directory == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{Error: "directory is required"})
		return
	}

	count, err := h.service.IndexDirectory(req.Directory)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIError{Error: err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message":   "indexed successfully",
		"chunks":    count,
		"directory": req.Directory,
	})
}

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	count, err := h.service.Stats()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIError{Error: err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"total_chunks": count,
	})
}

func (h *Handler) handleReindex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req indexRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{Error: "failed to read request body"})
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{Error: "invalid JSON"})
		return
	}

	if req.Directory == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{Error: "directory is required"})
		return
	}

	count, err := h.service.Reindex(req.Directory)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIError{Error: err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message":   "reindexed successfully",
		"chunks":    count,
		"directory": req.Directory,
	})
}

type teachRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func (h *Handler) handleTeach(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req teachRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{Error: "failed to read request body"})
		return
	}
	defer r.Body.Close()

	if err := json.Unmarshal(body, &req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{Error: "invalid JSON"})
		return
	}

	if req.Content == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{Error: "content is required"})
		return
	}

	count, err := h.service.Teach(req.Name, req.Content)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIError{Error: err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "taught successfully",
		"chunks":  count,
		"name":    req.Name,
	})
}

func (h *Handler) handleUpload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	file, header, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{Error: "file is required"})
		return
	}
	defer file.Close()

	if header.Size > 10<<20 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{Error: "file too large (max 10MB)"})
		return
	}

	tmpPath := filepath.Join(h.service.uploadDir, "rag_upload_"+header.Filename)
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIError{Error: "failed to save file"})
		return
	}
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, file); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIError{Error: "failed to save file"})
		return
	}
	tmpFile.Close()

	count, err := h.service.UploadFile(tmpPath)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIError{Error: err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "uploaded and indexed successfully",
		"chunks":  count,
		"file":    header.Filename,
	})
}
