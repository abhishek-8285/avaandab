package module

import (
	"fmt"
	"net/http"
)

// PaginationParams holds query parameters for paginated list views.
type PaginationParams struct {
	Query  string
	Status string
	Limit  int
	Page   int
	Offset int
}

// ParsePaginationParams extracts pagination and search parameters from an HTTP request.
func ParsePaginationParams(r *http.Request) PaginationParams {
	query := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")
	if status == "all" {
		status = ""
	}
	limit := 20
	_, _ = fmt.Sscanf(r.URL.Query().Get("limit"), "%d", &limit)
	if limit < 1 {
		limit = 20
	}
	page := 1
	_, _ = fmt.Sscanf(r.URL.Query().Get("page"), "%d", &page)
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit
	return PaginationParams{
		Query:  query,
		Status: status,
		Limit:  limit,
		Page:   page,
		Offset: offset,
	}
}

// PaginationData is passed to templates for rendering pagination controls.
type PaginationData struct {
	Page       int
	PerPage    int
	Total      int64
	TotalPages int
	HasPrev    bool
	HasNext    bool
	BasePath   string
}

// NewPaginationData computes pagination metadata from params and a total count.
func NewPaginationData(pp PaginationParams, total int64, basePath string) PaginationData {
	totalPages := int(total / int64(pp.Limit))
	if total%int64(pp.Limit) > 0 {
		totalPages++
	}
	if totalPages < 1 {
		totalPages = 1
	}
	return PaginationData{
		Page:       pp.Page,
		PerPage:    pp.Limit,
		Total:      total,
		TotalPages: totalPages,
		HasPrev:    pp.Page > 1,
		HasNext:    pp.Page < totalPages,
		BasePath:   basePath,
	}
}
